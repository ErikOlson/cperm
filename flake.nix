{
  description = "cperm — composable Claude Code permissions";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "cperm";
          version = "0.1.0";
          src = ./.;

          # After first build, run:
          #   nix build 2>&1 | grep 'got:'
          # and replace this with the actual hash
          vendorHash = null; # null means vendor directory is checked in

          meta = with pkgs.lib; {
            description = "Composable Claude Code permissions — Nix-inspired configuration composition";
            homepage = "https://github.com/erikmav/cperm";
            license = licenses.mit;
            mainProgram = "cperm";
          };
        };

        # Development shell with Go tooling
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            golangci-lint
            goreleaser
          ];
        };
      }
    );
}
