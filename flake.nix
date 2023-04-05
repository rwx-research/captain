{
  description = "A development environment for the Captain CLI";

  inputs.flake-utils.url = "github:numtide/flake-utils";
  inputs.nixpkgs.url =
    "github:nixos/nixpkgs/nixpkgs-unstable"; # Unstable is needed for Go 1.20

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
              (assertVersion "1.20.2" go)
              (assertVersion "1.52.2" golangci-lint)
              (assertVersion "1.14.0" mage)
            ];
        };
      });
}
