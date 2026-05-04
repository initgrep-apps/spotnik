FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
        bash zsh curl ca-certificates tar bats grep coreutils \
    && rm -rf /var/lib/apt/lists/*
RUN useradd -m -s /bin/zsh tester
USER tester
WORKDIR /home/tester
ENV HOME=/home/tester
ENV SHELL=/bin/zsh
RUN touch /home/tester/.zshrc && rm -f /home/tester/.bashrc
COPY --chown=tester:tester install.sh   /home/tester/install.sh
COPY --chown=tester:tester uninstall.sh /home/tester/uninstall.sh
COPY --chown=tester:tester tests/installer/bats /home/tester/bats
CMD ["bats", "/home/tester/bats"]
