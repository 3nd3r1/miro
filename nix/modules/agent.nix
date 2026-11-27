{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.miro.agent;
  miroPkgs = import ../packages.nix { inherit pkgs; };
  stateDir = builtins.dirOf cfg.statePath;
  inferenceEnabled = (config.miro.inference.enable or false);
in
{
  options.miro.agent = {
    enable = lib.mkEnableOption "Miro offline site reconciliation agent";

    siteId = lib.mkOption {
      type = lib.types.str;
      default = config.networking.hostName;
      description = "Site identifier verified against incoming bundle manifests.";
    };

    package = lib.mkOption {
      type = lib.types.package;
      default = miroPkgs.miro-agent;
      description = "Package providing the miro-agent binary.";
    };

    inboundDir = lib.mkOption {
      type = lib.types.str;
      default = "/var/lib/miro/transfer/inbound";
      description = "Directory where the courier drops incoming bundles.";
    };

    statusOutDir = lib.mkOption {
      type = lib.types.str;
      default = "/var/lib/miro/transfer/status-out";
      description = "Directory where the agent writes status bundles for the courier.";
    };

    statePath = lib.mkOption {
      type = lib.types.str;
      default = "/var/lib/miro/agent-state.json";
      description = "Path to the persisted agent reconciliation state JSON file.";
    };

    modelsDir = lib.mkOption {
      type = lib.types.str;
      default = "/var/lib/miro/models";
      description = "Content-addressed directory for GGUF model weights outside the Nix store.";
    };

    healthUrl = lib.mkOption {
      type = lib.types.str;
      default = "http://127.0.0.1:8080/health";
      description = "Local inference health endpoint probed after activation.";
    };

    healthTimeout = lib.mkOption {
      type = lib.types.str;
      default = "60s";
      description = "Maximum duration to wait for the local inference health endpoint.";
    };

    inferenceUnit = lib.mkOption {
      type = lib.types.str;
      default = "llama-cpp.service";
      description = "Systemd unit ensured active after system activation (empty to skip).";
    };
  };

  config = lib.mkIf cfg.enable {
    assertions = [
      {
        assertion = cfg.siteId != "";
        message = "miro.agent.siteId must be non-empty";
      }
      {
        assertion = !(lib.hasPrefix "/nix/store" cfg.modelsDir);
        message = "miro.agent.modelsDir must live outside /nix/store";
      }
    ];

    # Air-gapped site: disable all remote substituters.
    # Unsigned m-1 bundles use require-sigs = false; m-2 enables signature verification.
    system.switch.enable = true;

    nix.settings = {
      substituters = lib.mkForce [ ];
      trusted-substituters = lib.mkForce [ ];
      require-sigs = lib.mkDefault false;
      experimental-features = [ "nix-command" ];
    };

    systemd.tmpfiles.rules =
      [
        "d ${stateDir} 0755 root root -"
        "d /var/lib/miro/transfer 0755 root root -"
        "d ${cfg.inboundDir} 0755 root root -"
        "d ${cfg.statusOutDir} 0755 root root -"
      ]
      ++ lib.optional (!inferenceEnabled) "d ${cfg.modelsDir} 0755 root root -";

    systemd.services.miro-agent = {
      description = "Miro offline site reconciliation agent";
      after = [
        "local-fs.target"
        "systemd-tmpfiles-setup.service"
      ];
      wants = [ "nix-daemon.socket" ];
      path = [
        config.nix.package
        pkgs.systemd
        pkgs.coreutils
      ];
      restartIfChanged = false;
      stopIfChanged = false;
      serviceConfig = {
        Type = "oneshot";
        ExecStart = lib.escapeShellArgs [
          "${cfg.package}/bin/miro-agent"
          "reconcile"
          "--site"
          cfg.siteId
          "--inbound"
          cfg.inboundDir
          "--status-out"
          cfg.statusOutDir
          "--state"
          cfg.statePath
          "--models-dir"
          cfg.modelsDir
          "--health-url"
          cfg.healthUrl
          "--health-timeout"
          cfg.healthTimeout
          "--inference-unit"
          cfg.inferenceUnit
        ];
      };
    };

    systemd.paths.miro-agent = {
      description = "Watch Miro inbound bundle directory";
      wantedBy = [ "multi-user.target" ];
      pathConfig = {
        PathModified = cfg.inboundDir;
        MakeDirectory = true;
        Unit = "miro-agent.service";
      };
    };
  };
}
