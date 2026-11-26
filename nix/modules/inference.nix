{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.miro.inference;
  modelPath = "${cfg.modelsDir}/${cfg.modelSha256}";
in
{
  options.miro.inference = {
    enable = lib.mkEnableOption "Miro local CPU llama.cpp inference service";

    modelsDir = lib.mkOption {
      type = lib.types.str;
      default = "/var/lib/miro/models";
      description = "Directory containing content-addressed GGUF model files outside the Nix store.";
    };

    modelSha256 = lib.mkOption {
      type = lib.types.str;
      description = "SHA-256 hex digest of the GGUF model file under modelsDir.";
    };

    host = lib.mkOption {
      type = lib.types.str;
      default = "127.0.0.1";
      description = "Address for the local OpenAI-compatible inference server.";
    };

    port = lib.mkOption {
      type = lib.types.port;
      default = 8080;
      description = "Port for the local OpenAI-compatible inference server.";
    };

    extraSettings = lib.mkOption {
      type = lib.types.attrs;
      default = { };
      description = "Additional llama-server settings merged into services.llama-cpp.settings.";
    };
  };

  config = lib.mkIf cfg.enable {
    assertions = [
      {
        assertion = !(lib.hasPrefix "/nix/store" cfg.modelsDir);
        message = "miro.inference.modelsDir must live outside /nix/store";
      }
      {
        assertion = cfg.modelSha256 != "";
        message = "miro.inference.modelSha256 must be non-empty";
      }
    ];

    systemd.tmpfiles.rules = [
      "d /var/lib/miro 0755 root root -"
      "d ${cfg.modelsDir} 0755 root root -"
    ];

    services.llama-cpp = {
      enable = true;
      package = pkgs.llama-cpp;
      settings = lib.mkMerge [
        {
          host = cfg.host;
          port = cfg.port;
          model = modelPath;
          ctx-size = 2048;
          no-webui = true;
        }
        cfg.extraSettings
      ];
    };

    systemd.services.llama-cpp = {
      unitConfig.ConditionPathExists = modelPath;
      serviceConfig = {
        RestartSec = lib.mkForce 2;
      };
    };
  };
}
