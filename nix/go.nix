{ pkgs }:
{
  devShell = pkgs.mkShell {
    buildInputs = with pkgs; [
      go_1_25
      gopls
      gotools
      golangci-lint
      gotest
    ];

    packages = with pkgs; [
      hl-log-viewer
    ];

    shellHook = ''
      echo "Go version: $(go version)"
      echo "gopls version: $(gopls version)"
      echo "golangci-lint version: $(golangci-lint version)"
    '';
  };
}
