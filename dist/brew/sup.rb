require "language/go"

class Ssup < Formula
  desc "Super Stack Up. Super simple deployment tool remixed - think of it like 'make' for a network of servers, with a couple of batteries"
  homepage "https://github.com/momo182/ssup"
  url "https://github.com/pressly/sup/archive/4ee5083c8321340bc2a6410f24d8a760f7ad3847.zip"
  version "0.3.2"
  sha256 "7fa17c20fdcd9e24d8c2fe98081e1300e936da02b3f2cf9c5a11fd699cbc487e"

  depends_on "go"  => :build

  def install
    ENV["GOBIN"] = bin
    ENV["GOPATH"] = buildpath
    ENV["GOHOME"] = buildpath

    mkdir_p buildpath/"src/github.com/momo182/"
    ln_sf buildpath, buildpath/"src/github.com/momo182/sup"
    Language::Go.stage_deps resources, buildpath/"src"

    system "go", "build", "-o", bin/"ssup", "./cmd/ssup"
  end

  test do
    assert_equal "0.3", shell_output("#{bin}/bin/ssup")
  end
end
