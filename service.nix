inputs:
{
  config,
  pkgs,
  lib,
  ...
}:
let
  cfg = config.services.eko;
in
{
  meta.maintainers = with lib.maintainers; [ kyren223 ];

  options.services.eko = {
    enable = lib.mkEnableOption "eko service";

    package = lib.mkOption {
      description = "Eko package to use as the server executable";
      default = inputs.self.packages.${pkgs.system}.eko-server;
      type = lib.types.package;
    };

    openFirewall = lib.mkOption {
      description = "Open the ports in the firewall for the server.";
      default = false;
      type = lib.types.bool;
    };

    dataDir = lib.mkOption {
      description = "Eko data directory";
      default = "/var/lib/eko";
      type = lib.types.path;
    };

    logDir = lib.mkOption {
      description = "Eko logs directory";
      default = "/var/log/eko";
      type = lib.types.path;
    };

    tosFile = lib.mkOption {
      description = "Eko terms of service file";
      default = "/etc/eko/tos.md";
      type = lib.types.path;
    };

    privacyFile = lib.mkOption {
      description = "Eko privacy policy file";
      default = "/etc/eko/privacy.md";
      type = lib.types.path;
    };

    certFile = lib.mkOption {
      description = "Eko certificate key file";
      type = lib.types.path;
    };

    user = lib.mkOption {
      description = "Eko user";
      default = "eko";
      type = lib.types.str;
    };

    group = lib.mkOption {
      description = "Eko group";
      default = "eko";
      type = lib.types.str;
    };

    permission = lib.mkOption {
      description = "Permission to access dataDir";
      default = "0750";
      type = lib.types.str;
    };
  };

  config = lib.mkIf cfg.enable {
    # Open port 7223 for eko protocol, 443 for website
    networking.firewall.allowedTCPPorts = lib.mkIf cfg.openFirewall [ 7223 443 ];

    # Make sure eko user exists
    users.groups.eko = { };
    users.users.eko = {
      description = "Eko user";
      isNormalUser = true;
      createHome = true;
      home = cfg.dataDir;
      group = "eko";
    };

    systemd.tmpfiles.rules = [
      "d '${cfg.dataDir}' ${cfg.permission} ${cfg.user} ${cfg.group} - -"
      "z '${cfg.dataDir}' ${cfg.permission} ${cfg.user} ${cfg.group} - -"
    ];

    # Systemd service for eko
    systemd.services.eko = {
      description = "Eko - a secure terminal-native social media platform";

      wants = [ "network-online.target" ];
      after = [ "network-online.target" ];
      wantedBy = [ "multi-user.target" ];

      reloadTriggers = lib.mapAttrsToList (_: v: v.source or null) (
        lib.filterAttrs (n: _: lib.hasPrefix "eko/" n) config.environment.etc
      );

      environment = {
        EKO_SERVER_CERT_FILE = cfg.certFile;
        EKO_SERVER_LOG_DIR = cfg.logDir;
        EKO_SERVER_TOS_FILE = cfg.tosFile;
        EKO_SERVER_PRIVACY_FILE = cfg.privacyFile;
        USER = cfg.user;
        HOME = cfg.dataDir;
      };

      serviceConfig = {
        Restart = "on-failure";
        RestartSec = "10s";

        ExecStart = "/bin/sh -c 'ls && cd $HOME && ls && ${cfg.package}/bin/eko-server'";
        ExecReload = "${pkgs.coreutils}/bin/kill -SIGHUP $MAINPID";

        ConfigurationDirectory = "eko";
        # StateDirectory = "eko";
        # StateDirectoryMode = "0700";
        # Runtime directory and mode
        RuntimeDirectory = "eko";
        RuntimeDirectoryMode = cfg.permission;
        LogsDirectory = "eko";
        WorkingDirectory = cfg.dataDir;
        Type = "simple";

        User = cfg.user;
        Group = cfg.group;

        # Hardening
        ProtectHome = true;
        ProtectHostname = true;
        ProtectKernelLogs = true;
        ProtectKernelModules = true;
        ProtectKernelTunables = true;
        ProtectProc = "invisible";
        RestrictAddressFamilies = [
          "AF_INET"
          "AF_INET6"
          "AF_UNIX"
        ];
        RestrictNamespaces = true;
        RestrictRealtime = true;
        RestrictSUIDSGID = true;
        PrivateUsers = true;
        PrivateTmp = true;
        ProtectSystem = "strict";
        NoNewPrivileges = true;
      };
    };

  };
}
