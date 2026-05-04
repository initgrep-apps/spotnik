# archlinux only publishes amd64 manifests; pin the platform so this image
# can build on arm64 hosts (developer macs) via emulation. CI runners are
# already amd64, so the pin is a no-op there.
FROM --platform=linux/amd64 archlinux:base
# --disable-sandbox is needed when this image is built under x86_64 emulation
# (e.g. arm64 macs). Pacman's alpm sandbox user fails seccomp under qemu.
# On native amd64 CI runners the flag is harmless.
RUN pacman -Sy --noconfirm --disable-sandbox \
        bash curl ca-certificates tar bats grep coreutils diffutils sed gawk \
    && pacman -Scc --noconfirm
RUN useradd -m -s /bin/bash tester
USER tester
WORKDIR /home/tester
ENV HOME=/home/tester
RUN touch /home/tester/.bashrc
COPY --chown=tester:tester install.sh   /home/tester/install.sh
COPY --chown=tester:tester uninstall.sh /home/tester/uninstall.sh
COPY --chown=tester:tester tests/installer/bats /home/tester/bats
CMD ["bats", "/home/tester/bats"]
