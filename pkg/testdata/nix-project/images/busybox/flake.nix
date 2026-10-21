# Fixture for the nix build backend. The reference flake produces a busybox
# rootfs directory used by the discovery and build unit tests.
{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs";

  outputs = { self, nixpkgs }:
    {
      packages = nixpkgs.lib.genAttrs [ "x86_64-linux" "aarch64-linux" ] (system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        {
          default = pkgs.buildEnv {
            name = "busybox-rootfs";
            paths = [ pkgs.busybox ];
          };
        });
    };
}
