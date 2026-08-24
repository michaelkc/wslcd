{
  description = "wslcd - resolve Windows paths to /mnt paths for cd-ing in WSL";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
      ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        rec {
          wslcd = pkgs.buildGoModule {
            pname = "wslcd";
            # Flake builds come from git; tags are consumed via the .deb releases.
            version = "0.0.0+${self.shortRev or "dirty"}";
            src = ./src;
            vendorHash = null; # no Go module dependencies

            postInstall = ''
              install -D -m 0644 ${./packaging/wslcd.sh} $out/etc/profile.d/wslcd.sh
            '';

            meta = with pkgs.lib; {
              description = "Resolve Windows paths to /mnt paths for cd-ing in WSL";
              homepage = "https://github.com/michaelkc/wslcd";
              mainProgram = "wslcd";
            };
          };
          default = wslcd;
        }
      );
    };
}
