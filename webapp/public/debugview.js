window.addEventListener("load", function() {
    var ItemNames = {
        1: "おじいちゃん",
        2: "椅子畑",
        3: "椅子採掘場",
        4: "椅子工場",
        5: "禁断の秘術†椅子†",
        6: "確定拠出椅子",
        7: "クローン椅子技術",
        8: "椅子錬成陣",
        9: "太陽光発椅子",
        10: "加圧水型椅子炉",
        11: "椅子テラフォーミング",
        12: "椅子界への門",
        13: "椅子神託",
    }

    var getClickListener = function(idx) {
        return function() {
            game.buy(idx);
        };
    }

    var ItemRow = function(item_id) {
        this.item_id = item_id;
        var tr = document.createElement("tr");
        var attrs = [{
                "id": "b_name",
                "align": "center"
            },
            {
                "id": "b_n",
                "align": "right"
            },
            {
                "id": "b_button",
                "type": "button",
                "value": "建造"
            },
            {
                "id": "b_p",
                "align": "right"
            },
            {
                "id": "b_v",
                "align": "right"
            },
            {
                "id": "b_msg"
            },
        ];
        for (var i = 0; i < attrs.length; i++) {
            var td = document.createElement("td");
            if (i == 2) {
                var input = document.createElement("input");
                for (var k in attrs[i]) {
                    if (k === "id") input.setAttribute(k, attrs[i][k] + item_id);
                    else input.setAttribute(k, attrs[i][k]);
                }
                input.addEventListener("click", getClickListener(item_id));
                td.appendChild(input);
            } else {
                for (var k in attrs[i]) {
                    if (k === "id") td.setAttribute(k, attrs[i][k] + item_id);
                    else td.setAttribute(k, attrs[i][k]);
                }
            }
            tr.appendChild(td);
        }
        document.getElementById("b_table").appendChild(tr);
    }
    ItemRow.prototype.set = function(item, active) {
        document.getElementById("b_name" + this.item_id).textContent = ItemNames[this.item_id];
        document.getElementById("b_n" + this.item_id).textContent = item.count_built;
        if (item.disabled || !active) {
            document.getElementById("b_button" + this.item_id).setAttribute("disabled", "disabled");
        } else {
            document.getElementById("b_button" + this.item_id).removeAttribute("disabled");
        }
        document.getElementById("b_p" + this.item_id).textContent = item.next_price + " 脚";
        document.getElementById("b_v" + this.item_id).textContent = item.power + " 脚毎秒";
        var msg = "";
        for (var j = 0; j < item.building.length; j++) {
            msg += item.building[j].ordinal + " (" + item.building[j].dt + "ms後) ";
        }
        document.getElementById("b_msg" + this.item_id).textContent = msg;
    }
    var item_table = [];

    var debugIsu = function(n) {
        if (n.length <= 20) return n;
        return n[0] + "." + n.substr(1, 8) + "×10^" + (n.length - 1);
    }

    setInterval(function() {
        var state = game.getState();
        var active = game.isConnected();
        if (state === null) {
            // todo
        } else {
            document.getElementById("t_name").textContent = state.name;
            document.getElementById("t_x").textContent = state.getIsu() + " 脚";
            document.getElementById("t_v").textContent = state.getPower() + " 脚毎秒";
            var sending = state.getSending();
            var str_sending = "";
            for (var i = 0; i < sending.length; i++) {
                str_sending += debugIsu(sending[i].isu) + " (" + sending[i].dt + "ms後) ";
            }
            document.getElementById("t_sending").textContent = str_sending;
            var adding = state.getAdding();
            var str_adding = "";
            for (var i = 0; i < adding.length; i++) {
                str_adding += debugIsu(adding[i].isu) + " (" + adding[i].dt + "ms後) ";
            }
            document.getElementById("t_waiting").textContent = str_adding;
            if (active) {
                document.getElementById("game_button1").removeAttribute("disabled");
                document.getElementById("game_button2").removeAttribute("disabled");
                document.getElementById("game_button3").removeAttribute("disabled");
                document.getElementById("game_button4").removeAttribute("disabled");
            } else {
                document.getElementById("game_button1").setAttribute("disabled", "disabled");
                document.getElementById("game_button2").setAttribute("disabled", "disabled");
                document.getElementById("game_button3").setAttribute("disabled", "disabled");
                document.getElementById("game_button4").setAttribute("disabled", "disabled");
            }
            var numItem = state.getNumItem();
            while (item_table.length < numItem) {
                var item_id = item_table.length + 1;
                item_table.push(new ItemRow(item_id));
            }
            for (var i = 0; i < item_table.length; i++) {
                item_table[i].set(state.getItem(item_table[i].item_id), active);
            }
        }
    }, 55);

    document.getElementById("go_button").addEventListener("click", function() {
        game.start(document.getElementById("go_text").value);
    });
    document.getElementById("game_button1").addEventListener("click", function() {
        game.add(1);
    });
    document.getElementById("game_button2").addEventListener("click", function() {
        game.add(2);
    });
    document.getElementById("game_button3").addEventListener("click", function() {
        var x = document.getElementById("game_text3").value;
        if (!/^[1-9][0-9]*$/.test(x)) {
            document.getElementById("message").textContent = "error: 正整数でない";
            return;
        }
        document.getElementById("game_text3").value = "";
        game.addDirectly(x);
    });
    document.getElementById("game_button4").addEventListener("click", function() {
        var x = document.getElementById("game_text4_1").value;
        var y = document.getElementById("game_text4_2").value;
        if (!/^[1-9][0-9]*$/.test(x)) {
            document.getElementById("message").textContent = "error: 正整数でない";
            return;
        }
        if (!(y == "0" || /^[1-9][0-9]*$/.test(y))) {
            document.getElementById("message").textContent = "error: 非負整数でない";
            return;
        }
        y = +y;
        if (y > 100000) {
            document.getElementById("message").textContent = "error: 指数の上限は100000";
            return;
        }
        document.getElementById("game_text4_1").value = "";
        document.getElementById("game_text4_2").value = "";
        if (y > 0) {
            var r = "";
            var s = "0";
            for (;;) {
                if ((y & 1) == 1) r += s;
                y >>>= 1;
                if (y == 0) break;
                s += s;
            }
            x += r;
        }
        game.addDirectly(x);
    });

    game.setDebug();
});
