let
  pkgs = import <nixpkgs> { };
in
pkgs.mkShell {
  buildInputs = with pkgs; [
    unstable.go
    goimports
    goreleaser
  ];
}
