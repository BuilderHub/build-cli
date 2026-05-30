{
  description = "BuilderHub CLI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    nopher.url = "github:anthr76/nopher";
  };

  outputs = { self, nixpkgs, flake-utils, nopher }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        nopherLib = nopher.lib.${system};

        mkApp = pkgs: nopherLib: nopherLib.buildNopherGoApp {
          pname = "builderhub";
          version = "0.1.0";
          src = ./.;
          modules = ./nopher.lock.yaml;
          subPackages = [ "./cmd/builderhub" ];

          meta = {
            description = "BuilderHub platform CLI";
            mainProgram = "builderhub";
          };
        };

        app = mkApp pkgs nopherLib;
        linuxSystem =
          if system == "aarch64-darwin" then "aarch64-linux"
          else if system == "x86_64-darwin" then "x86_64-linux"
          else system;
        appLinux =
          if linuxSystem == system then app
          else mkApp nixpkgs.legacyPackages.${linuxSystem} nopher.lib.${linuxSystem};
      in
      {
        packages = {
          default = app;
          container = pkgs.dockerTools.buildLayeredImage {
            name = "builderhub";
            tag = "latest";
            contents = [ appLinux ];
            config = {
              Cmd = [ "${appLinux}/bin/builderhub" ];
            };
          };
        };

        apps.default = {
          type = "app";
          program = "${app}/bin/builderhub";
        };

        devShells.default = pkgs.mkShell {
          packages = [
            pkgs.go
            app
          ];
        };
      });
}
