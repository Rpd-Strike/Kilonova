{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  name = "kilonova";

  buildInputs = with pkgs; [
    yarn
    esbuild
  ];
}