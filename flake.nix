{
  description = "A development environment for the Captain CLI";

  inputs.flake-utils.url = "github:numtide/flake-utils";
  inputs.nixpkgs.url =
    "github:nixos/nixpkgs/nixos-unstable"; # Unstable is needed for Go 1.19

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = import nixpkgs { system = system; };
      in
      {
        formatter = pkgs.nixpkgs-fmt;
        devShell = pkgs.mkShell {
          # silence ginkgo deprecation warnings
          ACK_GINKGO_DEPRECATIONS = "2.6.0";

          packages = with pkgs;
            [ ginkgo go_1_19 golangci-lint mage ];
        };
      });
}
