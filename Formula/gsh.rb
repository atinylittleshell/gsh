# typed: false
# frozen_string_literal: true

# This file was generated by GoReleaser. DO NOT EDIT.
class Gsh < Formula
  desc "The Generative Shell"
  homepage "https://github.com/atinylittleshell/gsh"
  version "0.9.3"
  license "GPL-3.0"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/atinylittleshell/gsh/releases/download/v0.9.3/gsh_Darwin_x86_64.tar.gz"
      sha256 "2c17cd81ee6d6ca5995969cdc2874b627d9907b48399ad2ad28fad260ddf8e54"

      def install
        bin.install "gsh"
      end
    end
    if Hardware::CPU.arm?
      url "https://github.com/atinylittleshell/gsh/releases/download/v0.9.3/gsh_Darwin_arm64.tar.gz"
      sha256 "fdd747fd0a7fb391be6775bd804579b7735a632c0fbc5ecd4c8fd5547683f598"

      def install
        bin.install "gsh"
      end
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      if Hardware::CPU.is_64_bit?
        url "https://github.com/atinylittleshell/gsh/releases/download/v0.9.3/gsh_Linux_x86_64.tar.gz"
        sha256 "dd38e3e481086ba978f482dc95f111ca2d7b84b91a5e774ad58690dfce83e4d8"

        def install
          bin.install "gsh"
        end
      end
    end
    if Hardware::CPU.arm?
      if Hardware::CPU.is_64_bit?
        url "https://github.com/atinylittleshell/gsh/releases/download/v0.9.3/gsh_Linux_arm64.tar.gz"
        sha256 "a9d275d810adb50657949cd106adfbfee265330fcfd46fd57ec673d179077b47"

        def install
          bin.install "gsh"
        end
      end
    end
  end
end
