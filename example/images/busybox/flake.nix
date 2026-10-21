# Reference flake for the ContainerHive nix build backend. The default
# installable resolves to the mapped target system, so every image platform
# (linux/amd64, linux/arm64, ...) builds without per-platform flake inputs.
{
  description = "Busybox image rootfs built via the ContainerHive nix backend";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs";

  outputs = { self, nixpkgs }:
    {
      packages = nixpkgs.lib.genAttrs nixpkgs.lib.systems.flakeExposed (system:
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
