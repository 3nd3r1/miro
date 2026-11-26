{ pkgs, models }:
let
  model = models.qwen25-05b;
in
pkgs.testers.runNixOSTest {
  name = "site-a-smoke";

  nodes.site_a = {
    imports = [ ../sites/site-a.nix ];

    virtualisation = {
      vlans = [ ];
      qemu.networkingOptions = pkgs.lib.mkForce [ "-nic none" ];
      memorySize = 2048;
      cores = 2;
      diskSize = 2048;
    };

    environment.systemPackages = with pkgs; [
      curl
      jq
    ];
  };

  testScript = ''
    import json

    site_a.wait_for_unit("multi-user.target")

    # Confirm the site VM has no network interfaces beyond loopback.
    ifaces = site_a.succeed("ls /sys/class/net").strip().split()
    assert ifaces == ["lo"], f"expected only loopback interface, got {ifaces}"

    # Courier delivers model weights outside the Nix store.
    model_dest = "/var/lib/miro/models/${model.sha256}"
    site_a.copy_from_host("${model.file}", model_dest)
    site_a.succeed(f"chmod 0644 {model_dest}")

    # Start inference server and wait until the model is loaded and healthy.
    site_a.succeed("systemctl restart llama-cpp")
    site_a.wait_for_open_port(8080)
    site_a.wait_until_succeeds("curl -fsS http://127.0.0.1:8080/health")

    # Verify the local OpenAI-compatible chat completions endpoint responds.
    payload = json.dumps({
      "messages": [{"role": "user", "content": "Reply with the single word: PONG"}],
      "temperature": 0,
      "max_tokens": 16,
    })
    resp = site_a.succeed(
      f"curl -fsS -H 'Content-Type: application/json' -d '{payload}' http://127.0.0.1:8080/v1/chat/completions"
    )
    data = json.loads(resp)
    content = data["choices"][0]["message"]["content"].strip()
    print(f"llama-server response: {content!r}")
    assert len(content) > 0, f"expected non-empty completion, got {data}"
  '';
}
