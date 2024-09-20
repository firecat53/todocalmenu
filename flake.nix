{
  description = "Todocalmenu";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
  };

  outputs = {
    self,
    nixpkgs,
  }: let
    version = "0.5.2";
    systems = ["x86_64-linux" "i686-linux" "aarch64-linux"];
    forAllSystems = f:
      nixpkgs.lib.genAttrs systems (system:
        f rec {
          pkgs = nixpkgs.legacyPackages.${system};
          commonPackages = builtins.attrValues {
            inherit
              (pkgs)
              go
              ;
          };
        });
  in {
    devShells = forAllSystems ({
      pkgs,
      commonPackages,
    }: {
      default = pkgs.mkShell {
        packages = commonPackages ++ (with pkgs; [
              delve
              gnumake
              go-outline
              gocode-gomod
              godef
              golint
              gopkgs
              gopls
              gotools
            ]);
        shellHook = ''
          export GOPATH="$HOME/.local/src/go"
        '';
      };
    });
    packages = forAllSystems ({
      commonPackages,
      pkgs,
    }:{
      default = pkgs.buildGoModule {
        name = "todocalmenu";
        pname = "todocalmenu";
        inherit version;
        src = ./.;
        vendorHash = "sha256-DOJTdOyKEHZ2bZS1UisCWqNIuSDqrEUgd+jPjImTmgI=";
        proxyVendor = true;
        meta = {
          description = "Dmenu/Rofi launcher based management of iCalendar Todo lists";
          homepage = "https://github.com/firecat53/todocalmenu";
          license = pkgs.lib.licenses.mit;
          maintainers = ["firecat53"];
        };
      };
    });
  };
}
