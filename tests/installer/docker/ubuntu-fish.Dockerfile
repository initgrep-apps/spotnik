FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
        bash fish curl ca-certificates tar bats grep coreutils \
    && rm -rf /var/lib/apt/lists/*
RUN useradd -m -s /usr/bin/fish tester
USER tester
WORKDIR /home/tester
ENV HOME=/home/tester
ENV SHELL=/usr/bin/fish
# Pre-create fish config dir so install.sh's fish branch is taken.
RUN mkdir -p /home/tester/.config/fish/conf.d \
    && rm -f /home/tester/.bashrc /home/tester/.profile
COPY --chown=tester:tester install.sh   /home/tester/install.sh
COPY --chown=tester:tester uninstall.sh /home/tester/uninstall.sh
COPY --chown=tester:tester tests/installer/bats /home/tester/bats
CMD ["bats", "/home/tester/bats"]
