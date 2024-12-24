# nix run --file nix/fmt.nix
let
  treefmt-nix = builtins.getFlake "github:numtide/treefmt-nix/main";
  pkgs = import treefmt-nix.inputs.nixpkgs { };
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
