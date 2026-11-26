{ ... }:
{
  imports = [
    ../modules/inference.nix
  ];

  networking = {
    hostName = "site-a";
    useDHCP = false;
    interfaces = { };
    nameservers = [ ];
  };

  services.timesyncd.enable = false;

  miro.inference = {
    enable = true;
    modelSha256 = "74a4da8c9fdbcd15bd1f6d01d621410d31c6fc00986f5eb687824e7b93d7a9db";
  };

  system.stateVersion = "25.05";
}
