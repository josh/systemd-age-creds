{
  description = "A very basic flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/24.05";
  };

  outputs =
    { self, nixpkgs }:
    {
      packages = {
        aarch64-linux.systemd-age-creds =
          nixpkgs.legacyPackages.aarch64-linux.callPackage ./systemd-age-creds.nix
            { };
        x86_64-linux.systemd-age-creds =
          nixpkgs.legacyPackages.x86_64-linux.callPackage ./systemd-age-creds.nix
            { };

        aarch64-linux.default = self.packages.aarch64-linux.systemd-age-creds;
        x86_64-linux.default = self.packages.x86_64-linux.systemd-age-creds;
      };
    };
}
