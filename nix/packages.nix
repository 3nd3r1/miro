{ pkgs }:
let
  lib = pkgs.lib;

  goSrc = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.unions [
      ../go.mod
      ../cmd
      ../internal
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
  miro = mkGoBinary "miro" "cmd/miro";
  miro-agent = mkGoBinary "miro-agent" "cmd/miro-agent";
}
