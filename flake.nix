{
  description = ''
    A flake that provides multiple versions of HAProxy built in a
    similar way to OpenShift Ingress.";
  '';

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = { self, nixpkgs }: let
    supportedSystems = [ "aarch64-linux" "x86_64-linux" ];

    forAllSystems = f: builtins.listToAttrs (map (system: {
      name = system;
      value = f system;
    }) supportedSystems);

    # Function to create a specific version of the haproxy package.
    haproxyWithVersion = system: version: sha256:
    let
      pkgs = import nixpkgs { inherit system; };
      lib = nixpkgs.lib;
    in
    pkgs.haproxy.overrideAttrs (oldAttrs: rec {
      inherit version;

      buildFlags = [
        "CPU=generic"
        "TARGET=linux-glibc"
        "USE_CRYPT_H=1"
        "USE_GETADDRINFO=1"
        "USE_LINUX_TPROXY=1"
        "USE_OPENSSL=1"
        "USE_PCRE=1"
        "USE_REGPARM=1"
        "USE_ZLIB=1"
        "V=1"
      ];

      buildInputs = with pkgs; [
        libxcrypt
        openssl_1_1
        pcre
        zlib
      ];

      enableParallelBuilding = true;

      installPhase = ''
        mkdir -p $out/bin
        mv haproxy $out/bin/haproxy-${version}
      '';

      src = pkgs.fetchurl {
        url = "https://www.haproxy.org/download/${lib.versions.majorMinor version}/src/haproxy-${version}.tar.gz";
        inherit sha256;
      };
    });

    # List of versions and their respective sha256 hashes.
    versions = [
      { version = "2.2.0"; sha256 = "3c1a87160eea40e067f1e2813bfe692280a10c455beb17a8ee7fae11e4223274"; }
      { version = "2.2.1"; sha256 = "536552af1316807c01de727ad3dac84b3a2f5285db32e9bfdfe234e47ff9d124"; }
      { version = "2.2.2"; sha256 = "391c705a46c6208a63a67ea842c6600146ca24618531570c89c7915b0c6a54d6"; }
      { version = "2.2.3"; sha256 = "7209db363d4dbecb21133f37b01048df666aebc14ff543525dbea79be202064e"; }
      { version = "2.2.4"; sha256 = "87a4d9d4ff8dc3094cb61bbed4a8eed2c40b5ac47b9604daebaf036d7b541be2"; }
      { version = "2.2.5"; sha256 = "63ad1813e01992d0fbe5ac7ca3e516a53fc62cdb17845d5ac90260031b6dd747"; }
      { version = "2.2.6"; sha256 = "be1c6754cbaceafc4837e0c6036c7f81027a3992516435cbbbc5dc749bf5a087"; }
      { version = "2.2.7"; sha256 = "af8f46d9533b835bc3f02b6a769b0958077a7455e37f90ba4c86c7499cb243a7"; }
      { version = "2.2.8"; sha256 = "61f90e3e2a36bd8800a5bee31cba7eef37c9aa8a353b6c741edaa411510b14be"; }
      { version = "2.2.9"; sha256 = "21680459b08b9ba21c8cc9f5dbd0ee6e1842f57a3a67f87179871e1c13ebd4e3"; }
      { version = "2.2.10"; sha256 = "a027e9cd8f703ba48dc193f5ae34d9aa152221f67ab58a4e939c96b9f4edd3bc"; }
      { version = "2.2.11"; sha256 = "173a98506472bb1ba5a4a7f3a281125163f78bab5793bf6ba1f1d50853eb5f23"; }
      { version = "2.2.12"; sha256 = "6f30a3d70ec3993dd8a14bc1ab0d8043ec473aa8383c1185373c331f8883c61c"; }
      { version = "2.2.13"; sha256 = "9e3e51441c70bedfb494fc9d4b4d3389a71be9a3c915ba3d6f7e8fd9a57ce160"; }
      { version = "2.2.14"; sha256 = "6a9b702f04b07786f3e5878de8172a727acfdfdbc1cefe1c7a438df372f2fb61"; }
      { version = "2.2.15"; sha256 = "48cd0e1cb5de889657cd080fe66cc2ada8198cfece55d63f3a0c011553384cd9"; }
      { version = "2.2.16"; sha256 = "b81e03a181070d1f7bcdaeec9f09af701fb1ab015c3c9c997a881f4773e29375"; }
      { version = "2.2.17"; sha256 = "68af4aa3807d9c9f29b976d0ed57fb2e06acf7a185979059584862cc20d85be0"; }
      { version = "2.2.18"; sha256 = "a04a179d82711cac2199c4615621aaa9598c4788de2b34dccb9dd44da2862c50"; }
      { version = "2.2.19"; sha256 = "972e5a422dec3d9d01eb341eabd57d2d17d0e56e17d95d5c4c28b37b9c8aba12"; }
      { version = "2.2.20"; sha256 = "ef242bcb6d2cba3a16ebedbbb18c9919a75b7d98129ff7eb55c049e8ff742dd4"; }
      { version = "2.2.21"; sha256 = "a5827d1cd2f25dc31a603ee0494525ef59bb7e2012b1b1e09aad830bd17d0cb8"; }
      { version = "2.2.22"; sha256 = "823a12fdab61a4a547770e29ad3418de2d7d4a5542a51fb3277f74a6321eccfc"; }
      { version = "2.2.23"; sha256 = "de573b786b66ee2ccb9a79a237b0e91d3c27a24721474f9569d5878e8eb9d4fa"; }
      { version = "2.2.24"; sha256 = "0e807310dce3a5293d2454d9c1b71eb8d16472305b66f076b384b50858b1e7f9"; }
      { version = "2.2.25"; sha256 = "beb407eb08b2c697d115a18c00d6a023f7a0fb2cb99bff9c34c9dfd9d2c52f2b"; }
      { version = "2.2.26"; sha256 = "6a72a8667f2a5afb00ca34febf138ba6a1bf04c35bdb1b725d2e79a92f7cd959"; }
      { version = "2.2.27"; sha256 = "79583ce27ed8f3df0a415a761a6b7b014e3ea582e6c4a324a82e9c27e46217d6"; }
      { version = "2.2.28"; sha256 = "5734766c61ed177c5db6f1e5f40dd43cc539c22452c69d73a8a74ef5c8608235"; }
      { version = "2.2.29"; sha256 = "1e41f49674fbf5663b55c5f7919a7d05e480730653f2bcdec384b8836efc1fb0"; }
      { version = "2.3.0"; sha256 = "08badc59e037883f788f2c8f3358fa3e351aeaee826fd8a719f1e6252aff78fc"; }
      { version = "2.3.1"; sha256 = "8d3bf1252a5b60b21e9885c8d0d6d89e932d320c2977a6522aed6df81eefca4b"; }
      { version = "2.3.2"; sha256 = "99cb73bb791a2cd18898d0595e14fdc820a6cbd622c762f4ed83f2884d038fd5"; }
      { version = "2.3.3"; sha256 = "0671f7197e1218f262641bdfc97639ccdbe0cd7cd67d362fb05f04620cd7d971"; }
      { version = "2.3.4"; sha256 = "60148cdfedd6b19c401dbcd75ccd76a53c20bc76c49032ba32af98a0a5c495ed"; }
      { version = "2.3.5"; sha256 = "7924539530bbf555829c7f5886be0b7fcf8d9c8ffe0867b7010beb670abfbe4b"; }
      { version = "2.3.6"; sha256 = "6d4620e5da1d93ed75f229011216db81d5097b68bf175e309f2fab2890bba036"; }
      { version = "2.3.7"; sha256 = "31ba7acd0d78367c71b56e4a87c9f11cd235fc5602bc5b84690779120e0a305b"; }
      { version = "2.3.8"; sha256 = "2aa2691238dbe6360318673603aecd1041df19d55447172f8cd988780788159c"; }
      { version = "2.3.9"; sha256 = "77110bc1272bad18fff56b30f4614bcc1deb50600ae42cb0f0b161fc41e2ba96"; }
      { version = "2.3.10"; sha256 = "9946e0cfc83f29072b3431e37246221cf9d4a9d28a158c075714d345266f4f35"; }
      { version = "2.3.11"; sha256 = "299fbc30e9c60cf35b9a507d28fcf2536d714704c24f843b8bbaf416df124ff5"; }
      { version = "2.3.12"; sha256 = "684670d71ffe1ab3f0bfd1be0e55ea9d932732081dd01858b9e351c1a909be3e"; }
      { version = "2.3.13"; sha256 = "c651a44f1a1158085962ea64852ab163e0d21c5ef0ea4b2c5218728ed4dbe257"; }
      { version = "2.3.14"; sha256 = "047e5c65a018ea5061d16bb4eeb2f6e96b0d3f6ceb82d7ba96478549bbee062a"; }
      { version = "2.3.15"; sha256 = "b11e5411fd2473f47c32e35f6af5e5e25b03b94c3bc641f41b1ec987aba27b25"; }
      { version = "2.3.16"; sha256 = "7a26c8a58dd6be9c7f5e8c89d85b3c8ef4f9825109c0d5fc8ff56b7d6d254320"; }
      { version = "2.3.17"; sha256 = "27b4ca300ef2b4de0dbd3e8dc24f476467c17d740081c2bfab360ad788943a0b"; }
      { version = "2.3.18"; sha256 = "adc185c0321a15f36d73b374089a944b578ae7e132a4709ba1e82283b06d51c4"; }
      { version = "2.3.19"; sha256 = "a468c7eb91ae8b978fc603d77ab8f562018dac6f30fc00ce70e1c978a8859f12"; }
      { version = "2.3.20"; sha256 = "a71a0c7bd5e439cc57a014e919a6946027d4ac5ad21e7a2f8f2a933512e17b11"; }
      { version = "2.3.21"; sha256 = "15b51b24ccc3366db92ddb9ebf6880af2657842d67d1756ff1ac6d90151de668"; }
      { version = "2.4.0"; sha256 = "0a6962adaf5a1291db87e3eb4ddf906a72fed535dbd2255b164b7d8394a53640"; }
      { version = "2.4.1"; sha256 = "1b2458b05e923d70cdc00a2c8e5579c2fcde9df16bbed8955f3f3030df14e62e"; }
      { version = "2.4.2"; sha256 = "edf9788f7f3411498e3d7b21777036b4dc14183e95c8e2ce7577baa0ea4ea2aa"; }
      { version = "2.4.3"; sha256 = "ce479380be5464faa881dcd829618931b60130ffeb01c88bc2bf95e230046405"; }
      { version = "2.4.4"; sha256 = "116b7329cebee5dab8ba47ad70feeabd0c91680d9ef68c28e41c34869920d1fe"; }
      { version = "2.4.5"; sha256 = "adf4fdacd29ef0cb39dc7b79288f3e56c4bc651eeab0c73bb02ab9090943027b"; }
      { version = "2.4.6"; sha256 = "978370aa340fcc1f8bf986e600baeb724befd8284dfee3f3c2feb11f9e37aebb"; }
      { version = "2.4.7"; sha256 = "52af97f72f22ffd8a7a995fafc696291d37818feda50a23caef7dc0622421845"; }
      { version = "2.4.8"; sha256 = "e3e4c1ad293bc25e8d8790cc5e45133213dda008bfd0228bf3077259b32ebaa5"; }
      { version = "2.4.9"; sha256 = "d56c7fe3c5afedd1b9a19e1b7f8f954feaf50a9c2f205f99891043858b72a763"; }
      { version = "2.4.10"; sha256 = "4838dcc961a4544ef2d1e1aa7a7624cffdc4dda731d9cb66e46114819a3b1c5c"; }
      { version = "2.4.11"; sha256 = "bbebd025c6c960e147cb63d2cf9e3e84861d8220481519130a0ab4281e17b03c"; }
      { version = "2.4.12"; sha256 = "6984a94466739e5e8188949a3d1731634087226a12aada8bf6f81f9d316ca4f3"; }
      { version = "2.4.13"; sha256 = "4788fe975fe7e521746f826c25e80bc95cd15983e2bafa33e43bff23a3fe5ba1"; }
      { version = "2.4.14"; sha256 = "e6346b406b911b94c88eb05a5f622d53d49ffc247468fb03c12a4ffe3cc5ff04"; }
      { version = "2.4.15"; sha256 = "3958b17b7ee80eb79712aaf24f0d83e753683104b36e282a8b3dcd2418e30082"; }
      { version = "2.4.16"; sha256 = "8c5533779bb8125ef8dbd56a72b1d3fd47fa6bcdf2d257d3cc001269b059cee9"; }
      { version = "2.4.17"; sha256 = "416ca95d51bb57eaea0d6657c06a760faa63473dca10ac6f9e68b994088d73f4"; }
      { version = "2.4.18"; sha256 = "d7e46b56ac789d4fcf3ca209a109871e67ce4efca20a6537052f542f2af9616c"; }
      { version = "2.4.19"; sha256 = "99bd348a2b0ec58ce99510c4b6a2316e1f88137e59c4d8e0f39a2ecb3415a682"; }
      { version = "2.4.20"; sha256 = "415c62d2159941a17c443941aa85e7353047d79f3b72a0e7062473f7e753cacc"; }
      { version = "2.4.21"; sha256 = "ba525f1982c52fb72b25ab87a0e96292f415cc8f757412edff736606f5384cf7"; }
      { version = "2.4.22"; sha256 = "0895340b36b704a1dbb25fea3bbaee5ff606399d6943486ebd7f256fee846d3a"; }
      { version = "2.5.0"; sha256 = "16a5ed6256ca3670e41b76366a892b08485643204a3ce72b6e7a2d9a313aa225"; }
      { version = "2.5.1"; sha256 = "3e90790dfc832afa6ca4fdf4528de2ce2e74f3e1f74bed0d70ad54bd5920e954"; }
      { version = "2.5.2"; sha256 = "2de3424fd7452be1c1c13d5e0994061285055c57046b1cb3c220d67611d0da7e"; }
      { version = "2.5.3"; sha256 = "d6fa3c66f707ff93b5bd27ce69e70a8964d7b7078548b51868d47d7df3943fe4"; }
      { version = "2.5.4"; sha256 = "dc4015d85c7fef811b459803b763001d809b07a9251dc1864fedb9a07b44aefb"; }
      { version = "2.5.5"; sha256 = "063c4845cdb2d76f292ef44d9c0117a853d8d10ae5d9615b406b14a4d74fe4b9"; }
      { version = "2.5.6"; sha256 = "be4c71753f01af748531139bff3ade5450a328e7a5468c45367e021e91ec6228"; }
      { version = "2.5.7"; sha256 = "e29f6334c6bdb521f63ddf335e2621bd2164503b99cf1f495b6f56ff9f3c164e"; }
      { version = "2.5.8"; sha256 = "8477167a4785757f35f478a584c9a0aadd77122a2acbe7276aafaa53b69b03ac"; }
      { version = "2.5.9"; sha256 = "dc9a835c7026537419a311db64669e5782f32de49afc7b45f4277db62bc6b586"; }
      { version = "2.5.10"; sha256 = "ec590cde074e45786dcff9c8dbd72fe2f0848f171f27c2a377be4d013da4ec1f"; }
      { version = "2.5.11"; sha256 = "01bd318a149dab5a31afb89ad71d89e53b5fe1f8ec56677829e0b67d557de9e8"; }
      { version = "2.5.12"; sha256 = "e79f37e4e0d1cc0599ed965879c87e829e5b1bb6d17aa8aa2006cd20465dd214"; }
      { version = "2.5.13"; sha256 = "461a4037802d8ec7cf6005386616e9781bc3c0b5223588f48b776cb190e369a5"; }
      { version = "2.6.0"; sha256 = "90f8e608aacd513b0f542e0438fa12e7fb4622cf58bd4375f3fe0350146eaa59"; }
      { version = "2.6.1"; sha256 = "915b351e6450d183342c4cdcda7771eac4f0f72bf90582adcd15a01c700d29b1"; }
      { version = "2.6.2"; sha256 = "f9b7dc06e02eb13b5d94dc66e0864a714aee2af9dfab10fa353ff9f1f52c8202"; }
      { version = "2.6.3"; sha256 = "d18f7224a87b5cd6bbabb04d238f79a79fa461f0f8e1f257575cef8da2a307d9"; }
      { version = "2.6.4"; sha256 = "f07d67ada2ff3a999fed4e34459c0489536331a549665ac90cb6a8df91f4a289"; }
      { version = "2.6.5"; sha256 = "ce9e19ebfcdd43e51af8a6090f1df8d512d972ddf742fa648a643bbb19056605"; }
      { version = "2.6.6"; sha256 = "d0c80c90c04ae79598b58b9749d53787f00f7b515175e7d8203f2796e6a6594d"; }
      { version = "2.6.7"; sha256 = "cff9b8b18a52bfec277f9c1887fac93c18e1b9f3eff48892255a7c6e64528b7d"; }
      { version = "2.6.8"; sha256 = "a02ad64550dd30a94b25fd0e225ba699649d0c4037bca3b36b20e8e3235bb86f"; }
      { version = "2.6.9"; sha256 = "f01a1c5f465dc1b5cd175d0b28b98beb4dfe82b5b5b63ddcc68d1df433641701"; }
      { version = "2.6.10"; sha256 = "e71b2cd9ca1043345f083a5225078ccf824dced2b5779d86f11fa4e88f451773"; }
      { version = "2.6.11"; sha256 = "e0bc430ac407747b077bc88ee6922b4616fa55a9e0f3ec84438dfb055eb9a715"; }
      { version = "2.6.12"; sha256 = "58f9edb26bf3288f4b502658399281cc5d6478468bd178eafe579c8f41895854"; }
      { version = "2.7.0"; sha256 = "0f7bdebd9b0d7abfd89087bf36af6bd1520d3234266349786654e32e186b4768"; }
      { version = "2.7.1"; sha256 = "155f3a2fb6dfc1fdfd13d946a260ab8dd2a137714acb818510749e3ffb6b351d"; }
      { version = "2.7.2"; sha256 = "63bc6ec0302d0ebbe1fa769c19606640de834ac8cb07447b80799cb563dc0f3f"; }
      { version = "2.7.3"; sha256 = "b17e51b96531843b4a99d2c3b6218281bc988bf624c9ff90e19f0cbcba25d067"; }
      { version = "2.7.4"; sha256 = "84cb806030569e866812eed38ebd1a8ea6fe1d9800001e59924ec3dd39553b9f"; }
      { version = "2.7.5"; sha256 = "e2c6e43270c35a4009a70052d26c1ddb90b63a650f81305a748f229737a74502"; }
      { version = "2.7.6"; sha256 = "133f357ddb3fcfc5ad8149ef3d74cbb5db6bb4a5ab67289ce0b0ab686cdeb74f"; }
    ];

    # Define an overlay with multiple versions of the 'haproxy' package.
    multiHAProxyOverlay = system: self: super: builtins.listToAttrs (builtins.map (ver: {
      name = "haproxy_${builtins.replaceStrings ["."] ["_"] ver.version}";
      value = haproxyWithVersion system ver.version ver.sha256;
    }) versions);

    # Apply the overlay to nixpkgs.
    customPkgs = system: import nixpkgs {
      inherit system;
      # We need to use an overlay because we want to add multiple
      # versions of the same package to the package set. Normally,
      # each package has a unique name within the package set, and
      # there can only be one package with a given name. However, in
      # our case, we want to add multiple versions of the same package
      # (in this case, haproxy) with different version numbers. The
      # overlay applies to the nixpkgs package set, adding the new
      # package versions on top of the existing ones. This means that
      # we can access the multiple versions of haproxy by specifying
      # their version numbers in the package name. If we were to add
      # the packages directly to the packages attribute, we would need
      # to give each package a unique name, which would defeat the
      # purpose of having multiple versions of the same package. we
      # can still access the original haproxy package without
      # specifying a version number. The original haproxy package is
      # part of the nixpkgs package set, and it will be included in
      # the customPkgs package set we create when applying the
      # overlay.
      overlays = [ (multiHAProxyOverlay system) ];
      config = { };
    };

  in {
    # Expose the packages in the outputs.packages attribute
    packages = forAllSystems (system: let
      haproxyPackages = builtins.listToAttrs (builtins.map (ver: {
        name = "haproxy_${builtins.replaceStrings ["."] ["_"] ver.version}";
        value = (customPkgs system)."haproxy_${builtins.replaceStrings ["."] ["_"] ver.version}";
      }) versions);
    in {
      # Default is most latest release in the latest LTS series.
      default = haproxyPackages.haproxy_2_6_12;
    } // haproxyPackages);
  };
}
