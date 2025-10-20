{ buildGoModule }:

let
  src = ../../../.;
in
buildGoModule {
  pname = "quotes-app-go";
  version = "0.1.0";

  src = src;
  # This must be updated when `go.mod` or `go.sum` are changed.
  vendorHash = "sha256-AgRpj3B9fWoiTcZ5YO2gbW+NXiGHGA6HelMF0YpJ3WE=";
}
