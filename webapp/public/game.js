var game = (function(){
    var AddValueDelay = 1000;
    var BuyItemDelay = 1000;

    var unicodeMode = true;

    var getTime = function() {
        return new Date().getTime();
    }

    var Clock = function() {
        this.delta = null;
        this.time = null;
    }
    Clock.prototype.get = function(t) {
        if (this.delta === null) {
            return null;
        } else {
            return Math.max(this.time, getTime() + this.delta)
        }
    }
    Clock.prototype.set = function(t) {
        var y = this.get();
        this.delta = t - getTime();
        this.time = t;
    }
    var clock = new Clock();

    var exp2unicode = function(exp) {
        if (!unicodeMode) return "^" + exp;
        return (exp+'').replace(/[0-9]/g,function(i){return "⁰¹²³⁴⁵⁶⁷⁸⁹".charAt(i);});
    }

    var conv = function(a) {
        var m = a[0];
        var n = a[1];
        var x = 0;
        while (m / Math.pow(10, x) >= 1e+9) x++;
        m = Math.floor(m / Math.pow(10, x));
        n += x;
        if (n == 0) {
            return m + "";
        } else {
            var s = Math.max(1e8, Math.min(1e+9 - 1, m)) + "";
            return s[0] + "." + s.substr(1) + "×10" + exp2unicode(n + 8)
        }
    }
    var conv_exp = function(a) {
        var m = a[0];
        var n = a[1];
        m *= Math.pow(10, n);
        return Math.floor(Math.log10(m));
    }

    var add_xx = function(x, y) {
        if (x[1] < y[1]) return add_xx(y, x);
        if (x[1] === y[1]) return [x[0] + y[0], x[1]];
        return add_xx(x, [y[0] / Math.pow(10, x[1] - y[1]), x[1]]);
    }

    var getIsuDelta = function(a) {
        var m = a[0] / 10;
        var n = a[1];
        m *= Math.pow(10, n);
        var exp =  Math.floor(Math.log10(m));
        return Math.min(50, Math.max(0, exp));
    }

    var calcSchedule = function(schedule, time) {
        var k = null;
        for (var i = 0; i < schedule.length; i++) {
            if (schedule[i].time <= time) k = i;
        }
        if (k == null) return null;
        var tt = schedule[k].time;
        var vv = schedule[k].total_power;
        var xx = add_xx(schedule[k].milli_isu, [(time - tt) * vv[0], vv[1]]);
        return {"isu": conv([xx[0] / 1000, xx[1]]), "power": conv(vv),
                "isu_delta": getIsuDelta(vv), "isu_delta_unicode": exp2unicode(getIsuDelta(vv))};
    }

    var calcMapping = function(mapping, time) {
        var r = [];
        for (var t in mapping) {
            var dt = t - time;
            if (dt > 0) r.push({"isu": mapping[t], "dt": dt});
        }
        r.sort(function(a,b){ return a.dt - b.dt; });
        return r;
    }

    var calcDisabled = function(on_sale, item_id, time) {
        return !(item_id in on_sale && time >= on_sale[item_id]);
    }

    var calcItem = function(b, time, disabled) {
        var building = [];
        var k = null;
        var nn = b.count_built;
        for (var j = 0; j < b.building.length; j++) {
            if (b.building[j].time <= time) {
                k = j;
                nn = b.building[j].count_built;
            } else {
                for (var y = nn + 1; y <= b.building[j].count_built; y++) {
                    building.push({
                        "dt": b.building[j].time - time,
                        "ordinal": y
                    });
                }
            }
            nn = b.building[j].count_built;
        }
        return {
            "count_built": k != null ? b.building[k].count_built : b.count_built,
            "next_price": conv(b.next_price),
            "power": k != null ? conv(b.building[k].power) : conv(b.power),
            "disabled": disabled,
            "building": building,
        };
    }

    var GameState = function(name, data) {
        this.name = name;
        this._data_adding = {};
        for (var i = 0; i < data.adding.length; i++) {
            var b = data.adding[i];
            this._data_adding[b.time] = b.isu;
        }
        this._data_schedule = data.schedule;
        this._data_items = data.items;
        this._on_sale = {};
        for (var i = 0; i < data.on_sale.length; i++) {
            this._on_sale[data.on_sale[i].item_id] = data.on_sale[i].time;
        }
        this._isu = "0";
        this._power = "0";
        this._sending = [];
        this._adding = [];
        this._num_item = 0;
        this._items = {};
    }
    GameState.prototype.update = function(time, sending, buying) {
        var isu_power = calcSchedule(this._data_schedule, time);
        if (isu_power != null) {
            this._isu = isu_power.isu;
            this._power = isu_power.power;
            this._isu_delta = isu_power.isu_delta;
            this._isu_delta_unicode = isu_power.isu_delta_unicode;
        }
        this._sending = calcMapping(sending, time);
        this._adding = calcMapping(this._data_adding, time);
        this._num_item = this._data_items.length;
        this._items = {};
        for (var i = 0; i < this._data_items.length; i++) {
            var b = this._data_items[i];
            var item_id = b.item_id;
            var disabled = true;
            if (!buying) disabled = calcDisabled(this._on_sale, item_id, time);
            this._items[item_id] = calcItem(b, time, disabled);
        }
    }
    GameState.prototype.getIsu = function() {
        return this._isu;
    }
    GameState.prototype.getPower = function() {
        return this._power;
    }
    GameState.prototype.getIsuDelta = function() {
        return this._isu_delta;
    }
    GameState.prototype.getIsuDeltaUnicode = function() {
        return this._isu_delta_unicode;
    }
    GameState.prototype.getPowerExp = function() {
        return this._power_exp;
    }
    GameState.prototype.getSending = function() {
        return this._sending;
    }
    GameState.prototype.getAdding = function() {
        return this._adding;
    }
    GameState.prototype.getNumItem = function() {
        return this._num_item;
    }
    GameState.prototype.getItem = function(item_id) {
        if (item_id in this._items) {
            return this._items[item_id];
        }
        return null;
    }

    var Room = function(name) {
        this.name = name;
        this.conn = null;
        this.isOpen = false;
        this.reqCount = 0;
        this.callbacks = {};
        this.stateTime = null
        this.gameState = null;
        this.count_bought = null;
        this.sending = {};
        this.buying = false;
        this.addValue = [0, 0];
    }
    Room.prototype.connect = function(uri) {
        var self = this;
        self.conn = new WebSocket(uri);
        self.conn.onopen = function() {
            console.log("onopen");
            self.isOpen = true;
        }
        self.conn.onmessage = function(msg) {
            if (msg && msg.data) {
                var res = JSON.parse(msg.data);
                console.log(res);
                if (res.request_id) {
                    self.callbacks[res.request_id](res);
                    self.callbacks[res.request_id] = null;
                } else {
                    self.receiveData(res);
                }
            }
        }
        self.conn.onclose = function() {
            console.log("onclose");
            self.isOpen = false;
        }
        self.conn.onerror = function(err) {
            console.log("onerror", err);
        }
    }
    Room.prototype.receiveData = function(data) {
        if (this.stateTime == null || data.schedule[0].time >= this.stateTime) {
            this.stateTime = data.schedule[0].time;

            clock.set(data.time);

            this.count_bought = {};
            for (var i = 0; i < data.items.length; i++) {
                var b = data.items[i];
                this.count_bought[b.item_id] = b.count_bought;
            }

            this.gameState = new GameState(this.name, data);
        }
    }
    Room.prototype.sendRequest = function(req, callback) {
        var self = this;
        if (!self.isOpen) {
            console.log("not connected");
            return;
        }
        var c = ++self.reqCount;
        req.request_id = c;
        self.callbacks[c] = callback;
        self.conn.send(JSON.stringify(req));
    }
    Room.prototype.close = function() {
        this.conn.close();
    }
    Room.prototype.getAddValue = function() {
        if (this.addValue[1] === 0) {
            return this.addValue[0] + "";
        } else {
            var s = this.addValue[0] + "";
            while (s.length < 15) {
                s = "0" + s;
            }
            return (this.addValue[1] * 5) + s;
        }
    }
    Room.prototype.sendAdd = function(time, isu) {
        var self = this;

        var t = time;
        while (t in self.sending) t++;
        self.sending[t] = isu;
        self.sendRequest({
            "action": "addIsu",
            "time": t,
            "isu": isu,
        }, function(resp) {
            delete self.sending[t];
        });
    }
    Room.prototype.update = function() {
        var self = this;

        var addval = self.getAddValue();
        if (addval != "0") {
            self.addValue = [0, 0];
            var t = clock.get() + AddValueDelay;
            self.sendAdd(t, addval);
        }
    }
    Room.prototype.getState = function() {
        if (this.gameState == null) return null;
        this.gameState.update(clock.get(), this.sending, this.buying);
        return this.gameState;
    }
    Room.prototype.add = function(idx) {
        this.addValue[idx - 1] += 1;
    }
    Room.prototype.addDirectly = function(value) {
        this.sendAdd(clock.get() + AddValueDelay, value);
    }
    Room.prototype.buy = function(item_id) {
        var self = this;
        if (self.buying) return;
        self.buying = true;
        self.sendRequest({
            "action": "buyItem",
            "item_id": item_id,
            "time": clock.get() + BuyItemDelay,
            "count_bought": self.count_bought[item_id],
        }, function(resp) {
            self.buying = false;
        });
    }
    var room = null;

    var startGame = function(name) {
        if (room != null) room.close();
        room = null;

        var xhr = new XMLHttpRequest();
        xhr.responseType = 'json';
        xhr.open("GET", "/room/" + encodeURIComponent(name), true);
        xhr.onreadystatechange = function() {
            if (this.readyState == 4 && this.status == 200) {
                if (this.response) {
                    var host = this.response.host;
                    if (host === "") {
                        host = location.host;
                    }
                    var addr = "ws://" + host + this.response.path;
                    room = new Room(name);
                    room.connect(addr);
                }
            }
        }
        xhr.send();
    }

    return {
        "start": function(name) {
            startGame(name);
        },
        "getState": function() {
            if (room == null) return null;
            room.update();
            return room.getState();
        },
        "isConnected": function() {
            if (room == null) return false;
            return room.isOpen;
        },
        "add": function(type) {
            if (room != null) room.add(type);
        },
        "addDirectly": function(value) {
            if (room != null) room.addDirectly(value);
        },
        "buy": function(item_id) {
            if (room != null) room.buy(item_id);
        },
        "setDebug": function() {
            unicodeMode = false;
        }
    };
})();
