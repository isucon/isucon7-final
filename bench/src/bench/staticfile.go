package main

type StaticFile struct {
	Path string
	Size int64
	Hash string
}

var (
	StaticFiles = []*StaticFile{
		&StaticFile{"/favicon.ico", 1406, "d9084cb1d3e53ea5d96a7a878b0ac8e3"},
		&StaticFile{"/game.js", 11613, "4e8b3d31eba78dc570907460ced483be"},
		&StaticFile{"/gui.js", 14560, "ccb3f7de52fb178c8747d39a2888c002"},
		&StaticFile{"/images/5000_button.png", 11416, "eb09e073d616d1fc1123fef29b7ce952"},
		&StaticFile{"/images/background.png", 572, "a75af09c247626c017c5653a483c0a3b"},
		&StaticFile{"/images/bg_op.png", 73372, "fe3980fe501aa23ba4e230462efbad54"},
		&StaticFile{"/images/buybutton_background.png", 43132, "a6a433cbedad8203f56cd078475308f8"},
		&StaticFile{"/images/chair-2421605.png", 58610, "1233ac478f764b13c3ddb4a1a6f38c5c"},
		&StaticFile{"/images/icon_atom.png", 4470, "ba2cf57a1f73d33505bb7e1a7e07d933"},
		&StaticFile{"/images/icon_bluering.png", 4031, "95b90bfa0b740a1391a71f4ce50cc713"},
		&StaticFile{"/images/icon_chair.png", 2559, "29e02a1b487727bd2e2049c8c45581c4"},
		&StaticFile{"/images/icon_circle.png", 3475, "e68b61021b41597af21a931d72d908eb"},
		&StaticFile{"/images/icon_clone.png", 8063, "1bf31c3851e40caf9e9b1aaeb62da34e"},
		&StaticFile{"/images/icon_factory.png", 4263, "9087988e00064975ce6a2c2c565a4455"},
		&StaticFile{"/images/icon_farm.png", 4254, "29dc586b258a6ecc220133cf151fc18d"},
		&StaticFile{"/images/icon_jenkins.png", 3352, "59f467dd25a333797f199a90982c70ce"},
		&StaticFile{"/images/icon_mine.png", 2883, "1148b050ea3b08121f0e7de73bfc9c86"},
		&StaticFile{"/images/icon_reload.png", 3165, "08506cddf35857274130390244d10c3c"},
		&StaticFile{"/images/icon_solar.png", 4393, "2b3f789e3ebdb24a47f2ba9575aea91f"},
		&StaticFile{"/images/icon_sphere.png", 4235, "6080dce2852114225645702554540f26"},
		&StaticFile{"/images/icon_temple.png", 4558, "9093fcc9bee5c4193751e266e1eb0800"},
		&StaticFile{"/images/icon_torii.png", 3792, "4d67ed89bed180173ed84016bd6292ce"},
		&StaticFile{"/images/jenkins.png", 29020, "62c38a300ec97d401ea7fd1a4d948c18"},
		&StaticFile{"/images/line_h.png", 2917, "ced535c5871934feffcc623860e2082e"},
		&StaticFile{"/images/line_v.png", 6220, "58f276864637ac3788148a95cd89955c"},
		&StaticFile{"/images/stage_background.png", 740974, "8a62efa9ed6a1495fbf6b295ac62656b"},
		&StaticFile{"/images/stage_icon_atom.png", 2059, "28c471b125fc800c0089a0a37179d51d"},
		&StaticFile{"/images/stage_icon_candle.png", 1457, "cc9de7f8209ae88d45dec4aa80c6e1bc"},
		&StaticFile{"/images/stage_icon_circle.png", 3043, "978c773e6f055c7ab22afd1f9d73acb9"},
		&StaticFile{"/images/stage_icon_circle2.png", 2188, "7cca8260e2fef822fbb9119162ca304b"},
		&StaticFile{"/images/stage_icon_circle4.png", 2629, "3dfe3ed15b1bb7549c996491cb5ad1bf"},
		&StaticFile{"/images/stage_icon_clone.png", 2727, "e52e4864a2b61f25a6cf952d66b53486"},
		&StaticFile{"/images/stage_icon_factory.png", 1435, "155da95e86908674ad706a8c40e25737"},
		&StaticFile{"/images/stage_icon_farm.png", 2442, "05080f0825c6fde44af62d3ce8de73d1"},
		&StaticFile{"/images/stage_icon_gold.png", 2686, "c83098e7c9d3ecae72b3fa7a51c5412c"},
		&StaticFile{"/images/stage_icon_mine.png", 2960, "b69494f1aff2249e35b5e1ce12009cd9"},
		&StaticFile{"/images/stage_icon_solar.png", 2088, "cd53289dd6ffe9d868c460bc9e3baa6f"},
		&StaticFile{"/images/stage_icon_sphere.png", 2642, "f3ae863fdbb0c83e7361cb2cd7a8608f"},
		&StaticFile{"/images/stage_icon_temple.png", 2066, "4ea439cd3f9b5759fd7de387eba5f437"},
		&StaticFile{"/images/stage_icon_torii.png", 1424, "d92e806754cd6860b70e8036c070299b"},
		&StaticFile{"/images/stage_jenkins.png", 1779, "92b97ba2db0b959489a9c023ad0dbeae"},
		&StaticFile{"/phina.js", 308724, "8d949d77ab00a63b784c11712c64ab1a"},
	}
)
