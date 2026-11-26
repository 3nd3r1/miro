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

      goSrc = lib.fileset.toSource {
        root = ./.;
        fileset = lib.fileset.unions [
          ./go.mod
          ./cmd
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

      devShells.${system}.default = import ./shell.nix { inherit pkgs; };
    };
}
