{
  description = "dev-browser-mcp";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      systems = [ "aarch64-darwin" "x86_64-darwin" "aarch64-linux" "x86_64-linux" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f system);

      mkPackages = system:
        let
          pkgs = import nixpkgs { inherit system; };
          python = pkgs.python3.withPackages (ps: [ ps.playwright ]);
          src = self;
          mkWrapper = name: script: pkgs.writeShellApplication {
            inherit name;
            runtimeInputs = [ python pkgs.playwright-driver ];
            text = ''
              export PLAYWRIGHT_BROWSERS_PATH="${pkgs.playwright-driver.browsers}"
              export PYTHONPATH="${src}''${PYTHONPATH:+:$PYTHONPATH}"
              exec ${python}/bin/python -u ${src}/${script} "$@"
            '';
          };
          devBrowser = mkWrapper "dev-browser" "cli.py";
          devBrowserDaemon = mkWrapper "dev-browser-daemon" "daemon.py";
          devBrowserMcpServer = mkWrapper "dev-browser-mcp-server" "server.py";
          devBrowserSkill = pkgs.runCommand "dev-browser-skill" {} ''
            mkdir -p $out/dev-browser
            ln -s ${src}/SKILL.md $out/dev-browser/SKILL.md
          '';
        in
        {
          dev-browser = devBrowser;
          dev-browser-daemon = devBrowserDaemon;
          dev-browser-mcp-server = devBrowserMcpServer;
          dev-browser-skill = devBrowserSkill;
          default = devBrowser;
        };

      mkApps = system:
        let
          packages = mkPackages system;
        in
        {
          dev-browser = {
            type = "app";
            program = "${packages.dev-browser}/bin/dev-browser";
          };
          dev-browser-daemon = {
            type = "app";
            program = "${packages.dev-browser-daemon}/bin/dev-browser-daemon";
          };
          dev-browser-mcp-server = {
            type = "app";
            program = "${packages.dev-browser-mcp-server}/bin/dev-browser-mcp-server";
          };
          default = {
            type = "app";
            program = "${packages.dev-browser}/bin/dev-browser";
          };
        };
    in
    {
      packages = forAllSystems mkPackages;
      apps = forAllSystems mkApps;
    };
}
