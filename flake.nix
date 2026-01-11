{
  description = "dev-browser-mcp";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        python = pkgs.python3;
        pyPkgs = pkgs.python3Packages;
        version = "0.1.0";
        pythonPath = pyPkgs.makePythonPath [ pyPkgs.playwright ];

        devBrowser = pyPkgs.buildPythonApplication {
          pname = "dev-browser-mcp";
          inherit version;
          src = self;
          format = "other";
          nativeBuildInputs = [ pkgs.makeWrapper ];
          propagatedBuildInputs = [ pyPkgs.playwright pkgs.playwright-driver ];
          doCheck = false;

          installPhase = ''
            runHook preInstall

            site="$out/lib/${python.sitePackages}"
            mkdir -p "$site" "$out/libexec/dev-browser-mcp" "$out/bin"
            cp -r dev_browser_mcp "$site"/

            install -Dm755 cli.py "$out/libexec/dev-browser-mcp/cli.py"
            install -Dm755 daemon.py "$out/libexec/dev-browser-mcp/daemon.py"
            install -Dm755 server.py "$out/libexec/dev-browser-mcp/server.py"

            wrap() {
              local name="$1"
              local script="$2"
              makeWrapper ${python}/bin/python "$out/bin/$name" \
                --set PYTHONPATH "$site:${pythonPath}" \
                --set PLAYWRIGHT_BROWSERS_PATH "${pkgs.playwright-driver.browsers}" \
                --add-flags "-u $out/libexec/dev-browser-mcp/$script"
            }

            wrap dev-browser cli.py
            wrap dev-browser-daemon daemon.py
            wrap dev-browser-mcp-server server.py

            runHook postInstall
          '';

          meta = with pkgs.lib; {
            description = "Ref-based browser automation CLI + daemon + MCP server via Playwright";
            homepage = "https://github.com/joshp123/dev-browser-mcp";
            license = licenses.agpl3Plus;
            platforms = platforms.unix;
            mainProgram = "dev-browser";
          };
        };

        devBrowserSkill = pkgs.runCommand "dev-browser-skill" {} ''
          mkdir -p "$out/dev-browser"
          ln -s ${self}/SKILL.md "$out/dev-browser/SKILL.md"
        '';
      in {
        packages = {
          dev-browser = devBrowser;
          dev-browser-daemon = devBrowser;
          dev-browser-mcp-server = devBrowser;
          dev-browser-skill = devBrowserSkill;
          default = devBrowser;
        };

        apps = {
          dev-browser = { type = "app"; program = "${devBrowser}/bin/dev-browser"; };
          dev-browser-daemon = { type = "app"; program = "${devBrowser}/bin/dev-browser-daemon"; };
          dev-browser-mcp-server = { type = "app"; program = "${devBrowser}/bin/dev-browser-mcp-server"; };
          default = { type = "app"; program = "${devBrowser}/bin/dev-browser"; };
        };

        devShells.default = pkgs.mkShell {
          packages = [ python pyPkgs.playwright pkgs.playwright-driver ];
        };
      });
}
