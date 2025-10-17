{
  description = "A modern, POSIX-compatible, generative shell";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {inherit system;};
    in {
      defaultPackage = pkgs.buildGoModule rec {
        name = "gsh";
        version = "v${builtins.readFile ./VERSION}";
        src = pkgs.fetchFromGitHub {
          owner = "atinylittleshell";
          repo = "gsh";
          rev = version;
          hash = "sha256-r4vWse5zAzxaMNVXbISYHvB7158BF6MFWnVhJTN5Y0M=";
        };
        vendorHash = "sha256-Lcl6fyZf3ku8B8q4J4ljUyqhLhJ+q61DLj/Bs/RrQZo=";

        checkFlags = let
          # Skip tests that require network access or violate
          # the filesystem sandboxing
          skippedTests = [
            "TestReadLatestVersion"
            "TestHandleSelfUpdate_UpdateNeeded"
            "TestHandleSelfUpdate_NoUpdateNeeded"
            "TestFileCompletions"
          ];
        in ["-skip=^${builtins.concatStringsSep "$|^" skippedTests}$"];
      };
    });
}
