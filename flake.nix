{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    inputs:
    inputs.flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import inputs.nixpkgs { inherit system; };
        go = import ./nix/go.nix { inherit pkgs; };
        latex = import ./nix/latex.nix { inherit pkgs; };

      in
      {
        devShells = {
          default = go.devShell;
          docs = latex.devShell;
        };

        # devShells.default = pkgs.mkShell {
        #   buildInputs = with pkgs; [
        #     go_1_25
        #     gopls
        #     gotools
        #     golangci-lint
        #     gotest
        #   ];

        #   packages = with pkgs; [
        #     hl-log-viewer
        #   ];

        #   shellHook = ''
        #     echo "Go version: $(go version)"
        #     echo "gopls version: $(gopls version)"
        #     echo "golangci-lint version: $(golangci-lint version)"
        #   '';
        # };
      }
    );
}
# {
#   inputs = {
#     nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
#     flake-utils.url = "github:numtide/flake-utils";
#   };

#   outputs =
#     inputs:
#     inputs.flake-utils.lib.eachDefaultSystem (
#       system:
#       let
#         pkgs = import inputs.nixpkgs { inherit system; };
#       in
#       {
#         devShells.default = pkgs.mkShell {
#           buildInputs = with pkgs; [
#             go_1_25
#             gopls
#             gotools
#             golangci-lint
#             gotest
#           ];

#           packages = with pkgs; [
#             hl-log-viewer
#           ];

#           shellHook = ''
#             echo "Go version: $(go version)"
#             echo "gopls version: $(gopls version)"
#             echo "golangci-lint version: $(golangci-lint version)"
#           '';
#         };
#       }
#     );
# }
