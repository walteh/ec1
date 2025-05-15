package bootloader

var vsockrunner = /* bash */ `if mountpoint -q /sysroot && [ -d /sysroot/run/systemd/system ] \
 || mountpoint -q /root   && [ -d /root/run/systemd/system ] \
 || ( ! mountpoint -q /sysroot && ! mountpoint -q /root && [ -d /run/systemd/system ] ); then

  # We’re on a systemd guest → write out reparenting logic into the real root.

  # 1) Figure out where the real root is mounted:
  if mountpoint -q /sysroot; then
    REALROOT=/sysroot
  elif mountpoint -q /root; then
    REALROOT=/root
  else
    REALROOT=/
  fi

  # 2) SysV inittab fallback (for distros that still honor it):
  if [ -f "$REALROOT/etc/inittab" ]; then
    echo "vs:2345:respawn:/bin/vsock_guest_exec 2:1024 persistent" \
      >> "$REALROOT/etc/inittab"
  fi

  # 3) Drop in a systemd unit and enable it
  if [ -d "$REALROOT/etc/systemd/system" ]; then
    cat > "$REALROOT/etc/systemd/system/vsock-proxy.service" << 'EOF'
[Unit]
Description=VSock Guest Exec Proxy
After=network.target

[Service]
ExecStart=/bin/vsock_guest_exec 2:1024 persistent
Restart=always

[Install]
WantedBy=multi-user.target
EOF
    mkdir -p "$REALROOT/etc/systemd/system/multi-user.target.wants"
    ln -sf ../vsock-proxy.service \
      "$REALROOT/etc/systemd/system/multi-user.target.wants/vsock-proxy.service"
  fi

else
  # Non‑systemd or minimal initramfs → just background it now,
  # it’ll survive the exec/pivot without any further work.
  /bin/vsock_guest_exec 2:1024 persistent &
fi`
