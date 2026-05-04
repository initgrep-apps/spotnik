FROM fedora:40
RUN dnf install -y bash curl ca-certificates tar bats grep coreutils diffutils \
    && dnf clean all
RUN useradd -m -s /bin/bash tester
USER tester
WORKDIR /home/tester
ENV HOME=/home/tester
RUN touch /home/tester/.bashrc
COPY --chown=tester:tester install.sh   /home/tester/install.sh
COPY --chown=tester:tester uninstall.sh /home/tester/uninstall.sh
COPY --chown=tester:tester tests/installer/bats /home/tester/bats
CMD ["bats", "/home/tester/bats"]
