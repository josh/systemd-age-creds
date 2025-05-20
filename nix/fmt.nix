# nix run --file nix/fmt.nix
let
  system = builtins.currentSystem;
  nixpkgs = builtins.getFlake "github:NixOS/nixpkgs/nixos-25.05";
  pkgs = import nixpkgs { inherit system; };
  treefmt-nix = builtins.getFlake "github:numtide/treefmt-nix/main";
in
treefmt-nix.lib.mkWrapper pkgs {
  # keep-sorted start
  programs.actionlint.enable = true;
  programs.deadnix.enable = true;
  programs.gofmt.enable = true;
  programs.gofumpt.enable = true;
  programs.keep-sorted.enable = true;
  programs.nixfmt.enable = true;
  programs.prettier.enable = true;
  programs.statix.enable = true;
  # keep-sorted end
}
