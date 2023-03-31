{
  description = "A development environment for the Captain CLI";

  inputs.flake-utils.url = "github:numtide/flake-utils";
  inputs.nixpkgs.url =
    "github:nixos/nixpkgs/nixos-unstable"; # Unstable is needed for Go 1.19

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { system = system; };
        assertVersion = version: pkg: (
          assert (pkgs.lib.assertMsg (builtins.toString pkg.version == version) ''
            Expecting version of ${pkg.name} to be ${version} but got ${pkg.version};
          '');
          pkg
        );
      in
      {
        formatter = pkgs.nixpkgs-fmt;
        devShell = pkgs.mkShell {
          packages = with pkgs;
            [
              (assertVersion "2.9.2" ginkgo)
              (assertVersion "1.19.7" go_1_19)
              (assertVersion "1.52.2" golangci-lint)
              (assertVersion "1.14.0" mage)
            ];
        };
      });
}
