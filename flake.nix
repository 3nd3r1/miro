{
  description = "Apollo-style continuous delivery of LLM inference to air-gapped NixOS sites";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };
      lib = pkgs.lib;
      models = import ./nix/models.nix { inherit pkgs; };

      goSrc = lib.fileset.toSource {
        root = ./.;
        fileset = lib.fileset.unions [
          ./go.mod
          ./cmd
          ./internal
        ];
      };

      mkGoBinary =
        name: subPackage:
        pkgs.buildGoModule {
          pname = name;
          version = "0.1.0";
          src = goSrc;
          vendorHash = null;
          subPackages = [ subPackage ];
          env.CGO_ENABLED = 0;
          ldflags = [
            "-s"
            "-w"
            "-X main.version=0.1.0"
          ];
        };
    in
    {
      packages.${system} = {
        miro = mkGoBinary "miro" "cmd/miro";
        miro-agent = mkGoBinary "miro-agent" "cmd/miro-agent";
      };

      nixosModules.inference = ./nix/modules/inference.nix;

      nixosConfigurations.site-a = nixpkgs.lib.nixosSystem {
        inherit system;
        modules = [
          ./nix/sites/site-a.nix
          {
            fileSystems."/" = {
              device = lib.mkDefault "/dev/vda";
              fsType = lib.mkDefault "ext4";
            };
            boot.loader.grub.enable = lib.mkDefault false;
          }
        ];
      };

      checks.${system}.site-a-smoke = import ./nix/tests/smoke.nix {
        inherit pkgs models;
      };

      devShells.${system}.default = import ./shell.nix { inherit pkgs; };
    };
}
