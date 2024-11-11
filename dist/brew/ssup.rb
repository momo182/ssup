require "language/go"

class Ssup < Formula
  desc "Super Stack Up. Super simple deployment tool remixed - think of it like 'make' for a network of servers, with a couple of batteries"
  homepage "https://github.com/momo182/ssup"
  url "https://github.com/momo182/ssup/archive/c12c4ec52c35b832a992b876a6473e760b39c1a2.zip"
  version "0.3.2+mm"
  sha256 "c822f20990a24572b041da25ce7afacac078630306c3a0622775b603b7013989"

  depends_on "go"  => :build

  def install
    ENV["GOBIN"] = bin
    ENV["GOPATH"] = buildpath
    ENV["GOHOME"] = buildpath

    mkdir_p buildpath/"src/github.com/momo182/"
    ln_sf buildpath, buildpath/"src/github.com/momo182/ssup"
    Language::Go.stage_deps resources, buildpath/"src"

    system "go", "build", "-o", bin/"ssup", "./cmd/ssup"
  end

  test do
    assert_equal "0.3", shell_output("#{bin}/bin/ssup")
  end
end
