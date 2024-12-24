# nix run --file nix/lint.nix
let
  system = builtins.currentSystem;
  nixpkgs = builtins.getFlake "github:NixOS/nixpkgs/nixos-24.11";
  pkgs = import nixpkgs { inherit system; };
in
pkgs.writeShellApplication {
  name = "lint";
  runtimeInputs = with pkgs; [ golangci-lint ];
  text = ''
    exec golangci-lint run --config ${../.golangci.yml}
  '';
}
