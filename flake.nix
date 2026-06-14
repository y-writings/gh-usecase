{
  description = "gh-usecase CLI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      forEachSystem = f:
        nixpkgs.lib.genAttrs systems (system:
          f system (import nixpkgs { inherit system; })
        );
    in
    {
      packages = forEachSystem (_system: pkgs:
        let
          gh-usecase = pkgs.buildGoModule {
            pname = "gh-usecase";
            version = "0.0.0";

            src = pkgs.lib.cleanSourceWith {
              src = ./.;
              filter = path: _type:
                let
                  rel = pkgs.lib.removePrefix "${toString ./.}/" (toString path);
                in
                rel == "go.mod"
                || rel == "go.sum"
                || rel == "cmd"
                || rel == "internal"
                || pkgs.lib.hasPrefix "cmd/" rel
                || pkgs.lib.hasPrefix "internal/" rel;
            };
            subPackages = [ "cmd/gh-usecase" ];

            vendorHash = "sha256-+0MZl/+ybYYCiOdQXpEji6IDXuLAnbo1G1djX1sT3Uc=";

            ldflags = [
              "-s"
              "-w"
            ];

            meta = {
              description = "Small Go CLI for GitHub repository use cases";
              homepage = "https://github.com/y-writings/gh-usecase";
              license = pkgs.lib.licenses.mit;
              mainProgram = "gh-usecase";
            };
          };
        in
        {
          inherit gh-usecase;
          default = gh-usecase;
        });

      apps = forEachSystem (system: _pkgs:
        let
          gh-usecase = {
            type = "app";
            program = "${self.packages.${system}.gh-usecase}/bin/gh-usecase";
          };
        in
        {
          inherit gh-usecase;
          default = gh-usecase;
        });
    };
}
