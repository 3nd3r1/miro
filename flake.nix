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
      miroPackages = import ./nix/packages.nix { inherit pkgs; };
    in
    {
      packages.${system} = miroPackages;

      nixosModules = {
        agent = ./nix/modules/agent.nix;
        inference = ./nix/modules/inference.nix;
      };

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

      checks.${system} = {
        site-a-smoke = import ./nix/tests/smoke.nix {
          inherit pkgs models;
        };
        e2e-loop = import ./nix/tests/e2e.nix {
          inherit pkgs models miroPackages;
        };
      };

      devShells.${system}.default = import ./shell.nix { inherit pkgs; };
    };
}
