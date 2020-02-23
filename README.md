## STF IOS Support
### Build machine setup
1. Clone this repo down to your build machine
1. Install XCode
1. Add your developer Apple ID to XCode

    1. XCode -> XCode menu -> Preferences -> Accounts Tab
    1. `+` under `Apple IDs` list
    1. Choose `Apple ID`
    1. Login to your account so that dev certs can be downloaded
1. Run `./init.sh`

### Deploy server side:
1. On your STF server machine
    1. Pull STF server image `docker pull openstf/stf`
	1. Copy `docker-compose.yml` and `.env` from server/
	1. Generate certs for your system / domain
	1. Update `docker-compose.yml` cert paths and `.env`
	1. Start STF

		1. docker-compose up

### Build provider files:
1. Update config.json
1. Run `make dist`

    1. offline/dist.tgz will be created

### Deploy provider setup:
1. Copy `offline/dist.tgz` from build machine
1. Run `tar -xf dist.tgz`
1. Tweak `config.json` as desired

### Starting provider
1. Run `./run` ( and leave it running )
1. Plugin one or more IOS device(s)
1. Permissions dialog boxes appear; select accept for all of them
1. Device(s) shows up in STF with video and can be controlled. Yay

### Setting up with VPN
1. Install openvpn-server on your STF server machine
1. Create client certificate(s) using your favorite process...
1. Create ovpn file(s) with those client certs
1. Deploy those cert(s) to your provider machines; setting them up in Tunnelblick
1. Alter config.json on each provider to have the name of the cert setup in Tunnelblick
1. Start openvpn server on STF server
1. Start coordinator/provider on each provider machine

### Handling video not working
1. Run `./view_log -proc ffmpeg` to check for errors from ffmpeg
1. Run `./view_log -proc mirrorfeed` to check for errors from video feed process
1. Reboot your IOS device and try again

### Increase clicking speed
1. Jailbreak your IOS device
1. Install Veency through Cydia
1. Configure a VNC password if desired
1. Alter `config.json`

    1. Set `"use_vnc": true`
    1. Set `"vnc_scale": 2` ( or 3 depending on your device scale )
    1. If password used, set `"vnc_password": "[your password]"`
1. Start coordinator
1. Clicking is now nearly immediate!

### Debugging
1. run `./view_log` to see list of things that log
1. run `./view_log -proc [one from list]`


### Sample config.json


```
{
  "xcode_dev_team_id": "G34RY99M2F",
  "stf": {
    "ip": "192.168.1.56",
    "hostname": "tepmehatop"
  },
  "video": {
    "enabled": true,
    "use_vnc": true,
    "vnc_scale": 2,
    "vnc_password": "",
    "frame_rate": 10
  },
  "install": {
    "root_path": "/Users/imac/Desktop/iosSTF/stf_ios_support-master",
    "config_path": "/Users/imac/Desktop/iosSTF/stf_ios_support-master/config.json"
  },
  "vpn": {
    "type": "openvpn",
    "ovpn_config": "[path to your openvpn config file]"
  }
}


{
  "wda_folder": "./bin/wda",
  "xcode_dev_team_id": "G34RY99M2F",
  "network": {
    "coordinator_port": 8027,
    "video": "8000-8005",
    "dev_ios": "9240-9250",
    "vnc": "5901-5911",
    "wda": "8100-8105",
    "interface": "en0"
  },
  "stf": {
    "ip": "",
    "hostname": ""
  },
  "video": {
    "enabled": true,
    "use_vnc": false,
    "vnc_scale": 2,
    "vnc_password": "",
    "frame_rate": 5
  },
  "install": {
    "root_path": "",
    "config_path": ""
  },
  "log": {
    "main": "logs/coordinator",
    "proc_lines": "logs/procs",
    "wda_wrapper_stdout": "./logs/wda_wrapper_stdout",
    "wda_wrapper_stderr": "./logs/wda_wrapper_stderr"
  },
  "vpn": {
    "type": "openvpn",
    "ovpn_working_dir": "/usr/local/etc/openvpn",
    "tblick_name": ""
  },
  "bin_paths": {
    "wdaproxy": "bin/wdaproxy",
    "device_trigger": "bin/osx_ios_device_trigger",
    "video_enabler": "bin/osx_ios_video_enabler",
    "mirrorfeed": "bin/mirrorfeed",
    "openvpn": "/usr/local/opt/openvpn/sbin/openvpn",
    "iproxy": "/usr/local/bin/iproxy",
    "wdawrapper": "bin/wda_wrapper",
    "ffmpeg": "bin/ffmpeg"
  },
  "repos": {
    "stf": "https://github.com/nanoscopic/stf-ios-provider.git"  
  }
}
```
