# typed: false
# frozen_string_literal: true

# This file was generated by GoReleaser. DO NOT EDIT.
class Gsh < Formula
  desc "The Generative Shell"
  homepage "https://github.com/atinylittleshell/gsh"
  version "0.9.2"
  license "GPL-3.0"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/atinylittleshell/gsh/releases/download/v0.9.2/gsh_Darwin_x86_64.tar.gz"
      sha256 "b92f19bbc2d7862ae95185364831fbc7707aa8d9553a8b878a48642bc105bd73"

      def install
        bin.install "gsh"
      end
    end
    if Hardware::CPU.arm?
      url "https://github.com/atinylittleshell/gsh/releases/download/v0.9.2/gsh_Darwin_arm64.tar.gz"
      sha256 "b124d920a229a0dec46c8ea78bb7e3a75754fc200e449fd6cc163951e4a8ce88"

      def install
        bin.install "gsh"
      end
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      if Hardware::CPU.is_64_bit?
        url "https://github.com/atinylittleshell/gsh/releases/download/v0.9.2/gsh_Linux_x86_64.tar.gz"
        sha256 "98df722c027d1e63946e7a2511382f0448006156d295da25ef8c2198162634ff"

        def install
          bin.install "gsh"
        end
      end
    end
    if Hardware::CPU.arm?
      if Hardware::CPU.is_64_bit?
        url "https://github.com/atinylittleshell/gsh/releases/download/v0.9.2/gsh_Linux_arm64.tar.gz"
        sha256 "c24f1e5005586a74dbafdcf35b753df52f3a196a4a6d6ada70c32d0710c132ce"

        def install
          bin.install "gsh"
        end
      end
    end
  end
end
