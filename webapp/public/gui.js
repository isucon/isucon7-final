// グローバルに展開
phina.globalize();

// アセット
var ASSETS = {
  // 画像
  image: {
    'main_chair': 'images/chair-2421605.png',
    'background': 'images/background.png',
    'stage_background': 'images/stage_background.png',
    'bg_op': 'images/bg_op.png',
    'buybutton_background': 'images/buybutton_background.png',
    'line_v': 'images/line_v.png',
    'line_h': 'images/line_h.png',
    'icon_jenkins': 'images/icon_jenkins.png',
    'icon_factory': 'images/icon_factory.png',
    'icon_mine': 'images/icon_mine.png',
    'icon_temple': 'images/icon_temple.png',
    'icon_solar': 'images/icon_solar.png',
    'icon_bluering': 'images/icon_bluering.png',
    'icon_circle': 'images/icon_circle.png',
    'icon_sphere': 'images/icon_sphere.png',
    'icon_torii': 'images/icon_torii.png',
    'icon_chair': 'images/icon_chair.png',
    'icon_clone': 'images/icon_clone.png',
    'icon_atom': 'images/icon_atom.png',
    'icon_farm': 'images/icon_farm.png',
    'stage_icon_jenkins': 'images/stage_jenkins.png',
    'stage_icon_farm': 'images/stage_icon_farm.png',
    'stage_icon_mine': 'images/stage_icon_mine.png',
    'stage_icon_factory': 'images/stage_icon_factory.png',
    'stage_icon_circle': 'images/stage_icon_candle.png',
    'stage_icon_chair': 'images/stage_icon_gold.png',
    'stage_icon_clone': 'images/stage_icon_clone.png',
    'stage_icon_bluering': 'images/stage_icon_circle4.png',
    'stage_icon_solar': 'images/stage_icon_solar.png',
    'stage_icon_atom': 'images/stage_icon_atom.png',
    'stage_icon_sphere': 'images/stage_icon_sphere.png',
    'stage_icon_torii': 'images/stage_icon_torii.png',
    'stage_icon_temple': 'images/stage_icon_temple.png',
    'type2': 'images/5000_button.png',
  },
};

var SCREEN_WIDTH = 1360;
var SCREEN_HEIGHT = 1024;

var itemList = [
  {itemName: "おじいちゃん", iconName: 'icon_jenkins' },
  {itemName: "椅子畑", iconName: 'icon_farm' },
  {itemName: "椅子採掘場", iconName: 'icon_mine' },
  {itemName: "椅子工場", iconName: 'icon_factory' },
  {itemName: "禁断の秘術†椅子†", iconName: 'icon_circle' },
  {itemName: "確定拠出椅子", iconName: 'icon_chair' },
  {itemName: "クローン椅子技術", iconName: 'icon_clone' },
  {itemName: "椅子錬成陣", iconName: 'icon_bluering' },
  {itemName: "太陽光発椅子", iconName: 'icon_solar' },
  {itemName: "加圧水型椅子炉", iconName: 'icon_atom' },
  {itemName: "椅子テラフォーミング", iconName: 'icon_sphere' },
  {itemName: "椅子界への門", iconName: 'icon_torii' },
  {itemName: "椅子神託", iconName: 'icon_temple' },
];

phina.define('OpeningScene', {
  superClass: 'DisplayScene',

  init: function() {
    // 親クラス初期化
    this.superInit({
      width: SCREEN_WIDTH,
      height: SCREEN_HEIGHT,
    });

    var input = document.querySelector('#go_text');
    input.oninput = function() {
      form.text = input.value;
    };

    var background = Sprite('bg_op')
    .addChildTo(this)
    .setPosition(this.gridX.center(), this.gridY.center());
    background.width = this.width;
    background.height = this.height;

    var label = Label({
      x: this.gridX.center(),
      y: this.gridY.center(-5),
      text: 'Chair Constructor Online',
      fill: 'white',
      fontSize: 48,
      fontFamily: 'Merriweather',
    }).addChildTo(this);

    var default_text = 'Enter room id';
    var form = Button({
      width: 400,
      height: 80,
      text: default_text,
      fontSize: 50,
      fontFamily: 'Merriweather',
      fontColor: 'black',
      fill: 'white',
      stroke: 1,
      cornerRadius: 10,

    }).addChildTo(this)
      .setInteractive(true)
      .setPosition(this.gridX.center(), this.gridY.center());
    form.onpointstart = function() {
      // 最初にinputをクリックすると文字が消える
      if (this.text == default_text) {
        this.text = '';
      }
      input.focus();
    };

    var go_button = Button({
      x: this.gridX.center(),
      y: this.gridY.center(2),
      text: 'GO',
      fontFamily: 'Merriweather',
    }).addChildTo(this);

    var self = this;
    go_button.onpointstart = function() {
      var go_text = document.getElementById("go_text").value;
      if (go_text != '') {
        game.start(go_text);
        self.exit();
      }
    };
  },

});

phina.define('MainScene', {
  superClass: 'DisplayScene',

  init: function() {
    this.alreadyConnected = false;

    // 親クラス初期化
    this.superInit({
      width: SCREEN_WIDTH,
      height: SCREEN_HEIGHT,
    });
    this.backgroundColor = 'white';

    var in_animation = false;

    var background = Sprite('background')
    .addChildTo(this)
    .setPosition(this.gridX.center(), this.gridY.center());
    background.width = this.width;
    background.height = this.height;

    var stage_background = Sprite('stage_background')
    .addChildTo(this)
    .setPosition(this.gridX.center(), this.gridY.center());

    var line_v = Sprite('line_v')
    .addChildTo(this)
    .setPosition(this.gridX.span(5), this.gridY.center());
    line_v.height = this.height;

    var line_v2 = Sprite('line_v')
    .addChildTo(this)
    .setPosition(this.gridX.span(11), this.gridY.center());
    line_v2.height = this.height;

    var line_h = Sprite('line_h')
    .addChildTo(this)
    .setPosition(this.gridX.center(), this.gridY.span(0) + 4);
    line_h.width = this.width;

    var line_h2 = Sprite('line_h')
    .addChildTo(this)
    .setPosition(this.gridX.center(), this.gridY.span(16) - 4);
    line_h2.width = this.width;

    // 5000兆脚欲しいボタン
    var type2Button = Type2Button('type2')
      .addChildTo(this)
      .setPosition(this.gridX.span(2.5), this.gridY.span(13));

    var self = this;
    // スプライト画像作成
    var sprite = Sprite('main_chair')
    .addChildTo(this)
    .setPosition(this.gridX.span(2.5), this.gridY.center())
    .setInteractive(true)
    .on('pointstart', function(e) {
      isuDeltaExp = Math.min(10, Math.max(0, game.getState().getIsuDelta()));
      game.addDirectly("1"+"0".repeat(isuDeltaExp));
      var word = Label({
        x: e.pointer.x,
        y: e.pointer.y,
        text: (isuDeltaExp <= 8) ? "+"+Math.pow(10,isuDeltaExp) : "+10"+game.getState().getIsuDeltaUnicode(),
        fill: 'white',
        fontSize: 24,
        fontFamily: 'play',
      }).addChildTo(self);
      word.tweener.to({
        x:e.pointer.x,
        y:e.pointer.y - 250,
        alpha: 0,
      },2500,"swing").play().call(function () { this.remove() });
      if (!in_animation) {
        in_animation = true;
        this.tweener
        .scaleTo(1.02, 50)
        .scaleTo(1, 50)
        .scaleTo(1.02, 50)
        .scaleTo(1, 50)
        .call(function () {
          in_animation = false;
        })
        .play();
      }
    });

    // ラベル用の背景
    var rect = RectangleShape({
      x: 0,
      y: this.gridY.center(-5),
      width: this.gridX.width*10/16 - 17,
      height: this.gridY.width*3/16,
      fill: 'black',
    }).addChildTo(this);
    rect.alpha = 0.5;

    this.roomName = Label({
      x: this.gridX.span(2.5),
      y: this.gridY.center(-6),
      text: '',
      fill: 'white',
    }).addChildTo(this);

    // ラベルを生成
    this.label = Label({
      x: this.gridX.span(2.5),
      y: this.gridY.center(-5),
      text: '',
      fill: 'white',
      fontSize: 32,
      fontFamily: 'play',
    }).addChildTo(this);

    this.label_per_chair = Label({
      x: this.gridX.span(2.5),
      y: this.gridY.center(-4) - 10,
      text: '',
      fill: 'white',
      fontSize: 16,
      fontFamily: 'play',
    }).addChildTo(this);

    // エラーメッセージ表示エリア
    this.msgArea = ColoredLabel({
      x: this.gridX.span(13.5) + 4,
      y: this.gridY.span(15) - 4,
      width: this.gridX.width*5/16 - 16,
      height: this.gridY.width*2/16 - 16,
      text: "Connecting...\n",
      fill: 'red',
      background: 'black',
      fontSize: 24,
      fontFamily: 'play',
      align: 'left',
      verticalAlign: 'top'
    }).addChildTo(this);

    this.buyButtons = [];

    var ITEM_COUNT = 13;
    var DISPLAY_ICON_COUNT = 20;
    // stage上にiconが置かれているかを持っている配列
    this.stage_icons = new Array(ITEM_COUNT);
    for(var i = 0; i < ITEM_COUNT; i++) {
      this.stage_icons[i] = new Array(DISPLAY_ICON_COUNT);
      for(var j = 0; j < DISPLAY_ICON_COUNT; j++) {
        this.stage_icons[i][j] = false;
      }
    }
  },

  update: function () {
    var isConnected = game.isConnected();
    if (this.alreadyConnected && !isConnected) {
      this.msgArea.text = "ERROR: disconnected\n";
    } else if (!this.alreadyConnected && isConnected) {
      this.msgArea.text += "OK";
    }
    this.alreadyConnected = isConnected;
    var state = game.getState();
    if (state === null) {

    } else {
      this.label.text = state.getIsu() + " 脚";
      this.label_per_chair.text = state.getPower() + " 脚毎秒";
      this.roomName.text = "部屋名:" + state.name;

      // アイテムの追加および状態変更
      var numItem = state.getNumItem();
      while (this.buyButtons.length < numItem) {
        var item_id = this.buyButtons.length + 1;
        this.buyButtons.push(BuyButton({
          x: this.gridX.span(13.6),
          y: this.gridY.span(item_id) - 24,
          text: '',
          fill: 'black',
          itemName: itemList[item_id - 1].itemName,
          itemId: item_id,
          iconName: itemList[item_id - 1].iconName,
          domElement: 'b_button' + item_id,
        }).addChildTo(this));
      }
      for (i = 0; i < this.buyButtons.length; i++) {
        var buyButton = this.buyButtons[i];
        var item = state.getItem(buyButton.itemId);
        if (item.count_built > 0 &&
            item.count_built !== buyButton.label_num_owned.text) {
          buyButton.label_num_owned.text = item.count_built
        }
        if (item.next_price !== '' &&
            '💰 ' + item.next_price + ' 脚' !== buyButton.label_next_price.text) {
          buyButton.label_next_price.text = '💰 ' + item.next_price + " 脚";
        }
        if (item.disabled && !buyButton.disabled) {
          buyButton.disable();
        } else if (!item.disabled && buyButton.disabled) {
          buyButton.enable();
        }
      }

      // stageへのicon配置
      for (var i = 0; i < this.buyButtons.length; i++) {
        for (var j = 0; j < Number(this.buyButtons[i].label_num_owned.text); j++) {
          // ハミするともう見えない
          if (j >= 19) {
            break;
          }

          // まだ描画されていない場合描画
          var STAGE_BACKGROUND_HEIGHT = 78;
          if (this.stage_icons[i][j] == false) {
            var sprite = Sprite('stage_' + itemList[i].iconName).addChildTo(this);
            // 偶数・奇数でY軸をずらす
            if (j % 2 == 0) {
              sprite.setPosition(this.gridX.span(5.3+0.3*j), 30 + i * STAGE_BACKGROUND_HEIGHT);
            } else {
              sprite.setPosition(this.gridX.span(5.3+0.3*j), this.gridY.span(0.9) + i * STAGE_BACKGROUND_HEIGHT);
            }
            this.stage_icons[i][j] = true;
          }
        }
      }
    }
  },
});

// 5000兆脚欲しいボタン
phina.define('Type2Button', {
  superClass: 'Sprite',

  init: function(options) {
    this.superInit(options);

    this.setInteractive(true);

    this.on('pointstart', function(e) {
      console.log('clicked');
      game.add(2);
      var earned_chair = Label({
        x: e.pointer.x - this.x,
        y: e.pointer.y - this.y,
        text: '+5.0×10¹⁵',
        fill: 'white',
        fontSize: 40,
        fontFamily: 'play',
      }).addChildTo(this);
      earned_chair.tweener.to({
        x: e.pointer.x - this.x,
        y: e.pointer.y - this.y - SCREEN_HEIGHT/2,
        alpha: 0,
      },2500,"swing").play().call(function () { this.remove(); });

      this.tweener
        .scaleTo(1.5, 50)
        .scaleTo(1, 50)
        .scaleTo(1.5, 50)
        .scaleTo(1, 50)
        .play();
    });
  },
});

// 右下の接続ステータスボタン
phina.define('ColoredLabel', {
  superClass: 'RectangleShape',
  init: function(options) {
    this.superInit(options);
    this.fill = options.background;
    this.alpha = options.alpha;
    options.x = 8;
    options.y = 4;
    options.width -= 16;
    options.height -= 8;
    this._label = LabelArea(options).addChildTo(this);
  },
  _accessor: {
    text: {
      get: function() { return this._label.text; },
      set: function(v) { this._label.text = v; },
    },
  },
});

phina.define('BuyButton', {
  superClass: 'Button',

  init: function(options) {
    this.superInit(options);
    this.itemId = options.itemId;
    this.width = SCREEN_WIDTH/16*5; // 425
    this.height = 64;
    this.cornerRadius = 0;
    this.domElement = options.domElement;
    this.on('pointstart', function() {
      game.buy(options.itemId);
    });
    this.background = Sprite('buybutton_background').addChildTo(this);
    this.label_item_name = Label({
      x: 0,
      y: -8,
      text: options.itemName,
      fontSize: 24,
    }).addChildTo(this);
    this.label_num_owned = Label({
      x: this.width/2 - 18,
      y: 0,
      text: '',
      fill: '#404040',
      align: 'right',
      fontSize: 32,
      fontFamily: 'Merriweather',
    }).addChildTo(this);
    this.label_next_price = Label({
      x: 0,
      y: this.height/2 - 16,
      text: '',
      fontSize: 16,
      fontFamily: 'play',
    }).addChildTo(this);

    var icon = Sprite(options.iconName).addChildTo(this);
    icon.x = - (this.width/2 - 48 + 15);

    this.disable();
  },

  disable: function () {
    this.disabled = true;
    this.setInteractive(false);
    this.background.alpha = 0.5;
    this.label_item_name.fill = 'silver';
    this.label_next_price.fill = 'orange';
  },

  enable: function () {
    this.disabled = false;
    this.setInteractive(true);
    this.background.alpha = 1;
    this.label_item_name.fill = 'white';
    this.label_next_price.fill = 'lime';
  },
});

// メイン
phina.main(function() {
  var app = GameApp({
    width: SCREEN_WIDTH,
    height: SCREEN_HEIGHT,

    startLabel: 'opening_scene',
    assets: ASSETS,

    scenes: [
      {
        label: 'opening_scene',
        className: 'OpeningScene',
      },
      {
        label: 'main_scene',
        className: 'MainScene',
      },
    ]
  });

  app.enableStats();
  app.run();
});

