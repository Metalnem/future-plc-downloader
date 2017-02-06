package main

type magazine struct {
	name      string
	appKey    string
	secretKey string
	id        string
}

var magazines = []magazine{
	{"3D World", "LAVJyNS_Se-3CwbzRtrUlA", "42fb37c41af55dcf4590afcefcd4f1a4", "3DWorldMagazine"},
	{"APC Australia", "i6YbGG5r43qlIGhM2eNUhqwjhnm8cltL", "OJbBdFnYzI2ChBevlnb3oEstVBgE8fn1", "apcmag"},
	{"Australian T3", "vzR3YQcRQS6aWn4IkZg7Gg", "28c0579d513200cea0378b5319248893", "AustralianT3"},
	{"Comic Heroes", "7rYf2gwWQ5CWHaq8uaiWWQ", "c93cb7db4dd07da3ea6a8343cbad481d", "comicheroesmagazine"},
	{"Computer Arts", "er7z5PYeTK6kiin1ps_wzg", "06c15d1e9c50fc47645396de48461adf", "computerartsmagazine"},
	{"Computer Music Magazine", "2LYEqLQZTmWlJO20ZHQO0A", "51e19022b496ff733cde1f3885e51dc8", "computermusicmagazine"},
	{"Crime Scene", "kXLm815RyixPTWOLOnt7NfenrIJpUigV", "woafeHokZLxG9vrNKg4FbfmdSG2y4vk1", "crimescene"},
	{"Digital Camera World", "rkNkeSPyQumn78fjaGdv_Q", "1115db4ad36e0bdc7b8630df7f16028a", "DigitalcameraWorld"},
	{"Edge", "RymlyxWkRBKjDKsG3TpLAQ", "b9dd34da8c269e44879ea1be2a0f9f7c", "edgemagazine"},
	{"Future Music", "5_pl4TStQsi5VOdyAeIL8A", "10eadee2bbfa1fba350a41de28c00316", "futuremusicmagazine"},
	{"GamesMaster", "gKAnhmAQSEyK2Gwrf44JLg", "516ff640ab3d6405676ff288b8bebcee", "gamesmastermagazine"},
	{"GD Legacy", "jHsd9gOyRd2bvNCmdOtmXA", "5c74d313bd15f36535a63aa6666aae6f", "guitaristDELUXE"},
	{"Guitar Techniques", "2vKIh_6hSTilV_4mLD8n2g", "7eb456cc7339dd7e8e0c3b608f451f78", "guitartechniquesmagazine"},
	{"Guitar World Magazine", "IOzwl9bNQzmkoIw3PRBPyA", "6d517bad83ef09aebca1fe304b9367ed", "guitarworldpadmagazine"},
	{"Guitarist", "7e7_MyhPTdWaMiQ4ofrACw", "80e79d2ccddb263de9cb5e8da723340c", "GITmonthly"},
	{"ImagineFX", "EGqwsSIgQ0mA_sEz2ZPvgA", "a5203ae150bec3a5cd00625f05ee0ec9", "imaginefxmagazine"},
	{"iPad User", "ZxGqbTdBS0StEH3rhyCeJg", "ee6b72e1257cf356c685ed4f5d48d649", "ipadusermag"},
	{"Linux Format", "DiNG0P_pQnmKtCgvuCIyvQ", "db96a2cc010f0c09f59282c84080bcbe", "linuxformatmagazine"},
	{"Mac Life", "eZoIdr2PT2qfM6UOXbV6uw", "j2V3K70211H35J8", "maclifeipadmagazine"},
	{"MacFormat", "4li6co3IRoS5-vwcQBlHVg", "94f3880e9b7ba8b94038b36d25c85e7d", "macformatmagazine"},
	{"Maximum PC", "O4Eib8kfQ_umkERUOT7zwQ", "097036b27b47b60804bd57a244974b5e", "MaximumPCipadmagazine"},
	{"Minecraft Mayhem", "Oj9jKJDY8McwevhuZqT9eH3QqB2ePgeD", "CcagpkZG5jxaezryx84OVazFlAspXOaR", "MinecraftMayhem"},
	{"N-Photo", "gMyMneSlTN-Gto8n0_LGfQ", "8c13173ebc36bc18637b86127a567c0e", "nphoto"},
	{"net magazine", "rrE7fa_LTuqLn_WtUEXgrg", "80dc20862bec2127bc41b1fbef286808", "netmagazine"},
	{"Official Xbox Magazine (US Edition)", "BUi-yPGiQCOOLI03OW022A", "f50334dd99b2f6dbba55e237963d9308", "oxmUSAipadmagazine"},
	{"Paint & Draw", "SguLCNbRRztxRIkd8CwIjVzcuEwkZ8HB", "IJZ6TRdJngOu0ViUfUbdUdJqledMn1U3", "paintanddraw"},
	{"PC Format Legacy", "qX0qM8IMSrGLc16SbLW3Ww", "21f03ac1ffb3c857105e4d0e91c6249c", "pcformatmagazine"},
	{"PC Gamer (UK)", "SVK7cLVtRm2No1xQ3LBGhA", "4ri183rq5pEy04x", "pcgamermagazine"},
	{"PC Gamer (US)", "myI5-WslSQaGpIppM0EIsw", "1c8c84d00d04db96f98b124019395a88", "PCGamerUSAipadmagazine"},
	{"Photography Week", "5KaijsEVTCOIAs8nITGAOg", "f4d3c95ffac1960c3b8f798f1f541ac0", "photographyweek"},
	{"PhotoPlus", "ruM_6_SkSPOIWonMaSIqsA", "e239e189bc9d61056d732f4a242e195d", "photoplusmagazine"},
	{"Pi User Magazine", "U2EjNVP6R6ju16v7nyUhmRgRG3MTMu1b", "Wo6huYiCOiarsvt7XMtdQeYyUzrAitTl", "piuser"},
	{"Practical Photoshop", "0OQ2GXpKSlyyKipcGLrqGg", "51cdbab13707ddbc0b6c088598dd332a", "practicalphotoshopmagazine"},
	{"Professional Photography Magazine", "I9jtZjKCE51qzLJKCzzozG4VsLecxMuO", "1Q1maKYQ11wbMFhZ1DoUdVNj3ionnuao", "prophotography"},
	{"Rhythm", "y5PZ04QxQIWb63blvt2PsA", "ef591ab5a83fc3b60454f28bebff0c3a", "rhythmmagazine"},
	{"SFX", "JSWr6Lq2R1asm54Prcjfag", "19b87a09ac8f6838ab42b2d3fad70c32", "sfxmagazine"},
	{"T3", "qcY8QuyYS327FU8QJXImbw", "0c3cd05357c3b0eb15665769ac2790d1", "t3magazine"},
	{"TechLife Australia", "LPcgvEwgMdEQsqfjOwD7nJBBiBnjSHOL", "0FIcDR04qwqAhTJ74J5h444Ax3T4Vols", "techlifeaustralia"},
	{"Total Film", "VQzXImWMRZSfiNR0qVThHg", "1cd2829bb1d0b516e70e3b835d6effbb", "totalfilmmagazine"},
	{"Total Guitar", "y49rHJvTTuyo7RKVa8I7Aw", "dc1595f1bc7933327df39674c1163377", "totalguitarmagazine"},
	{"Windows Help & Advice", "vKx686oTRx6f_Ujd9-POuQ", "0acc0597c77414c7cc1219c8ce9307de", "officialwindowsmagazine"},
}
