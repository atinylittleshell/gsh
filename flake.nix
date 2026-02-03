{
  description = "A battery-included, POSIX-compatible, generative shell";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { nixpkgs, ... }:
  let
    forAllSystems = f:
      nixpkgs.lib.genAttrs
        [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ]
        (system: f nixpkgs.legacyPackages.${system});
  in {
    packages = forAllSystems (pkgs: {
      default = pkgs.buildGoModule rec {
        name = "gsh";
        version = "v1.3.3";
        src = pkgs.fetchFromGitHub {
          owner = "atinylittleshell";
          repo = "gsh";
          rev = version;
          hash = "sha256-kyEWFoBXuR23wM4Y17tcPmPLpcSKUXy8v857CYeyv0U=";
        };
        vendorHash = "sha256-0ZzdlcI6ZdaWq9yutdrONMkshwfoiHxmLupNXo8Zjtc=";

        nativeBuildInputs = with pkgs; [
          which
        ];

        # Skip tests that require network access or violate
        # the filesystem sandboxing. Basically all tests tries
        # to create a /homeless-shelter directory and errors with
        # 'read-only file system'.
        doCheck = false;
      };
    });
  };
}
