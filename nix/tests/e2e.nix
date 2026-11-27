{
  pkgs,
  models,
  miroPackages,
}:
let
  model = models.qwen25-05b;

  siteABase = {
    imports = [ ../sites/site-a.nix ];

    # QEMU VM tests boot via direct kernel/initrd without a GRUB boot disk.
    boot.loader.grub.enable = pkgs.lib.mkForce false;

    virtualisation = {
      vlans = [ ];
      qemu.networkingOptions = pkgs.lib.mkForce [ "-nic none" ];
      memorySize = 2048;
      cores = 2;
      diskSize = 6144;
      writableStore = true;
      writableStoreUseTmpfs = false;
    };

    environment.systemPackages = with pkgs; [
      curl
      jq
    ];
  };
in
pkgs.testers.runNixOSTest {
  name = "e2e-loop";

  nodes = {
    site_a = {
      imports = [ siteABase ];
      environment.etc."miro-release".text = "v1\n";
    };

    # Target generation closure built with identical VM test modules so
    # switch-to-configuration keeps the test backdoor and filesystems intact.
    site_a_v2 = {
      imports = [ siteABase ];
      environment.etc."miro-release".text = "v2\n";
      miro.inference.extraSettings.ctx-size = pkgs.lib.mkForce 1024;
    };

    hub =
      { nodes, ... }:
      {
        virtualisation = {
          memorySize = 1024;
          cores = 2;
          diskSize = 6144;
          writableStore = true;
          writableStoreUseTmpfs = false;
          additionalPaths = [
            nodes.site_a_v2.system.build.toplevel
            model.file
          ];
        };

        nix.settings = {
          experimental-features = [
            "nix-command"
            "flakes"
          ];
        };

        environment.systemPackages = [
          miroPackages.miro
          pkgs.jq
        ];
      };
  };

  testScript =
    { nodes, ... }:
    let
      siteAV2Path = "${nodes.site_a_v2.system.build.toplevel}";
    in
    ''
      import datetime
      import json
      import os
      import shutil
      from pathlib import Path

      site_a_v2 = "${siteAV2Path}"
      model_file = "${model.file}"
      model_name = "${model.name}"
      model_sha256 = "${model.sha256}"
      hub_state = "/var/lib/miro-hub/state.json"
      out_root = Path(os.environ["out"])

      hub.start()
      site_a.start()
      hub.wait_for_unit("multi-user.target")
      site_a.wait_for_unit("multi-user.target")
      site_a.wait_for_unit("miro-agent.path")

      # Confirm site-a has no network interfaces beyond loopback and boots on v1.
      ifaces = site_a.succeed("ls /sys/class/net").strip().split()
      assert ifaces == ["lo"], f"expected only loopback interface, got {ifaces}"
      assert site_a.succeed("cat /etc/miro-release").strip() == "v1"
      site_a.fail(f"test -e /var/lib/miro/models/{model_sha256}")

      # 1. Hub exports an unsigned bundle for site-a v2.
      hub.succeed(
        f"miro export --state {hub_state} --system {site_a_v2} "
        f"--model {model_file} --model-name {model_name} "
        f"site-a /var/lib/miro-hub/outbound/site-a-seq-0001"
      )
      before_status = hub.succeed(f"miro status --state {hub_state}")
      assert "desired: seq=1" in before_status, before_status
      assert "actual:  none" in before_status, before_status
      assert "site-a-seq-0001(seq=1)" in before_status, before_status

      # 2. Courier pulls the bundle off the hub onto the host transfer staging area.
      hub.copy_from_machine("/var/lib/miro-hub/outbound/site-a-seq-0001", "courier/outbound")
      bundle_host_dir = out_root / "courier" / "outbound" / "site-a-seq-0001"
      assert (bundle_host_dir / "manifest.json").is_file()

      # 3. Partial copy test: deliver cache/ and manifest.json without models/<sha256>.
      site_a.copy_from_host(str(bundle_host_dir / "cache"), "/var/lib/miro/transfer/inbound/cache")
      site_a.copy_from_host(
        str(bundle_host_dir / "manifest.json"),
        "/var/lib/miro/transfer/inbound/manifest.json",
      )

      # miro-agent.path triggers miro-agent, which must defer the incomplete bundle.
      site_a.wait_until_succeeds(
        "journalctl -u miro-agent.service --no-pager | grep -q 'deferring incomplete bundle'"
      )
      site_a.wait_until_succeeds(
        "test \"$(systemctl show -p ActiveState --value miro-agent.service)\" = inactive"
      )
      assert site_a.succeed("cat /etc/miro-release").strip() == "v1"
      site_a.fail("test -e /var/lib/miro/agent-state.json")
      site_a.fail("test -e /var/lib/miro/transfer/status-out/status.json")

      # 4. Courier finishes copying the remaining model weights into inbound/.
      # Stage outside inbound/ and move into inbound/models so miro-agent.path fires once complete.
      site_a.copy_from_host(str(bundle_host_dir / "models"), "/var/lib/miro/transfer/.models-staging")
      site_a.succeed("mv /var/lib/miro/transfer/.models-staging /var/lib/miro/transfer/inbound/models")

      # Clean up the large outbound bundle from $out so the check output stays small.
      shutil.rmtree(out_root / "courier" / "outbound")

      # 5. Wait for miro-agent.path -> miro-agent.service to activate v2 and write status.
      site_a.wait_until_succeeds(
        "test -f /var/lib/miro/transfer/status-out/status.json",
        timeout=datetime.timedelta(seconds=180),
      )
      site_a.wait_until_succeeds(
        "test \"$(systemctl show -p ActiveState --value miro-agent.service)\" != activating",
        timeout=datetime.timedelta(seconds=60),
      )
      state = site_a.succeed("systemctl show -p ActiveState --value miro-agent.service").strip()
      assert state == "inactive", f"miro-agent.service ended in state {state!r}"

      # Verify site-a is now running generation v2 and serving llama.cpp.
      assert site_a.succeed("cat /etc/miro-release").strip() == "v2"
      current_sys = site_a.succeed("readlink -f /run/current-system").strip()
      assert current_sys == site_a_v2, f"expected {site_a_v2}, got {current_sys}"
      site_a.succeed(f"test -f /var/lib/miro/models/{model_sha256}")
      site_a.wait_for_open_port(8080)
      site_a.wait_until_succeeds("curl -fsS http://127.0.0.1:8080/health")

      payload = json.dumps({
        "messages": [{"role": "user", "content": "Reply with the single word: PONG"}],
        "temperature": 0,
        "max_tokens": 16,
      })
      resp = site_a.succeed(
        f"curl -fsS -H 'Content-Type: application/json' -d '{payload}' http://127.0.0.1:8080/v1/chat/completions"
      )
      completion = json.loads(resp)["choices"][0]["message"]["content"].strip()
      assert len(completion) > 0, f"expected non-empty completion, got {resp}"

      # 6. Courier carries status-out/ back to the hub.
      site_a.copy_from_machine("/var/lib/miro/transfer/status-out", "courier/status")
      status_host_dir = out_root / "courier" / "status" / "status-out"
      assert (status_host_dir / "status.json").is_file()
      hub.copy_from_host(str(status_host_dir), "/var/lib/miro-hub/status-in/site-a")

      # 7. Hub ingests status and reports site-a updated and healthy.
      hub.succeed(f"miro import-status --state {hub_state} /var/lib/miro-hub/status-in/site-a")
      after_status = hub.succeed(f"miro status --state {hub_state}")
      print(after_status)
      assert "site: site-a" in after_status, after_status
      assert f"desired: seq=1 system={site_a_v2} model={model_sha256}" in after_status, after_status
      assert f"system={site_a_v2} model={model_sha256} health=healthy (ok)" in after_status, after_status
      assert "pending: none" in after_status, after_status

      status_json = json.loads(hub.succeed(f"miro status --state {hub_state} --json"))
      actual = status_json["sites"]["site-a"]["actual"]
      assert actual["last_applied_sequence"] == 1, actual
      assert actual["current_system"] == site_a_v2, actual
      assert actual["current_model_sha256"] == model_sha256, actual
      assert actual["health"]["healthy"] is True, actual
      assert actual["current_generation"] >= 1, actual
    '';
}
