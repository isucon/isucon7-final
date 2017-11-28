require_relative '../game'

RSpec.describe Game do
  describe '.calc_status' do
    let(:current_time) { 0 }
    let(:mitems) { {} }
    let(:addings) { [] }
    let(:buyings) { [] }

    subject { described_class.calc_status(current_time, mitems, addings, buyings) }

    context 'status empty' do
      it do
        expect(subject.adding).to be_empty
        expect(subject.schedule.length).to eq(1)
        expect(subject.on_sale).to be_empty
      end

      it do
        expect(subject.schedule[0].time).to eq(0)
        expect(subject.schedule[0].milli_isu).to satisfy do |exp|
          exp.mantissa == 0 && exp.exponent == 0
        end
        expect(subject.schedule[0].total_power).to satisfy do |exp|
          exp.mantissa == 0 && exp.exponent == 0
        end
      end
    end

    # 椅子が増える
    context 'status add' do
      let(:addings) do
        [
          described_class::Adding.new(nil, 100, '1'),
          described_class::Adding.new(nil, 200, '2'),
          described_class::Adding.new(nil, 300, '1234567890123456789'),
        ]
      end

      context do
        it do
          expect(subject.adding.length).to eq(3)
          expect(subject.schedule.length).to eq(4)
        end

        it do
          expect(subject.schedule[0].time).to eq(0)
          expect(subject.schedule[0].milli_isu).to satisfy do |exp|
            exp.mantissa == 0 && exp.exponent == 0
          end
          expect(subject.schedule[0].total_power).to satisfy do |exp|
            exp.mantissa == 0 && exp.exponent == 0
          end
        end

        it do
          expect(subject.schedule[1].time).to eq(100)
          expect(subject.schedule[1].milli_isu).to satisfy do |exp|
            exp.mantissa == 1000 && exp.exponent == 0
          end
          expect(subject.schedule[1].total_power).to satisfy do |exp|
            exp.mantissa == 0 && exp.exponent == 0
          end
        end

        it do
          expect(subject.schedule[2].time).to eq(200)
          expect(subject.schedule[2].milli_isu).to satisfy do |exp|
            exp.mantissa == 3000 && exp.exponent == 0
          end
          expect(subject.schedule[2].total_power).to satisfy do |exp|
             exp.mantissa == 0 && exp.exponent == 0
          end
        end

        it do
          expect(subject.schedule[3].time).to eq(300)
          expect(subject.schedule[3].milli_isu).to satisfy do |exp|
            exp.mantissa == 123456789012345 && exp.exponent == 7
          end
          expect(subject.schedule[3].total_power).to satisfy do |exp|
            exp.mantissa == 0 && exp.exponent == 0
          end
        end
      end

      context do
        let(:current_time) { 500 }

        it do
          expect(subject.adding.length).to eq(0)
          expect(subject.schedule.length).to eq(1)
        end

        it do
          expect(subject.schedule[0].time).to eq(500)
          expect(subject.schedule[0].milli_isu).to satisfy do |exp|
            exp.mantissa == 123456789012345 && exp.exponent == 7
          end
          expect(subject.schedule[0].total_power).to satisfy do |exp|
            exp.mantissa == 0 && exp.exponent == 0
          end
        end
      end
    end

    # 試しに１個買う
    context 'status buy single' do
      let(:initial_isu) { '10' }
      let(:mitems) do
        {
          1 => described_class::MItem.new(
            item_id:1,
            power1: 0,
            power2: 1,
            power3: 0,
            power4: 10,
            price1: 0,
            price2: 1,
            price3: 0,
            price4: 10
          )
        }
      end
      let(:addings) do
        [
          described_class::Adding.new(nil, 0, initial_isu)
        ]
      end
      let(:buyings) do
        [
          described_class::Buying.new(nil, 1, 1, 100)
        ]
      end

      it do
        expect(subject.adding.length).to eq(0)
        expect(subject.schedule.length).to eq(2)
        expect(subject.items.length).to eq(1)
      end

      it do
        expect(subject.schedule[0].time).to eq(0)
        expect(subject.schedule[0].milli_isu).to satisfy do |exp|
          exp.mantissa == 0 && exp.exponent == 0
        end
        expect(subject.schedule[0].total_power).to satisfy do |exp|
          exp.mantissa == 0 && exp.exponent == 0
        end
      end

      it do
        expect(subject.schedule[1].time).to eq(100)
        expect(subject.schedule[1].milli_isu).to satisfy do |exp|
          exp.mantissa == 0 && exp.exponent == 0
        end
        expect(subject.schedule[1].total_power).to satisfy do |exp|
          exp.mantissa == 10 && exp.exponent == 0
        end
      end
    end

    # 購入時間を見ます
    context 'on sale' do
      let(:current_time) { 1 }
      let(:mitems) do
        {
          1 => described_class::MItem.new(
            item_id: 1,
            power1: 0,
            power2: 1,
            power3: 0,
            power4: 1,
            price1: 0,
            price2: 1,
            price3: 0,
            price4: 1
          )
        }
      end
      let(:addings) do
        [
          described_class::Adding.new(nil, 0, '1')
        ]
      end
      let(:buyings) do
        [
          described_class::Buying.new(nil, 1, 1, 0)
        ]
      end

      it do
        expect(subject.adding.length).to eq(0)
        expect(subject.schedule.length).to eq(1)
        expect(subject.on_sale.length).to eq(1)
        expect(subject.items.length).to eq(1)
      end

      it do
        expect(subject.schedule[0]).to satisfy do |schedule|
          schedule.time == 1
        end
      end

      it do
        expect(subject.on_sale[0]).to satisfy do |on_sale|
          on_sale.item_id == 1 && on_sale.time == 1000
        end
      end

      it do
        expect(subject.items[0].count_bought).to eq(1)
        expect(subject.items[0].power).to satisfy do |exp|
          exp.mantissa = 1 && exp.exponent == 0
        end
        expect(subject.items[0].count_built).to eq(1)
        expect(subject.items[0].next_price).to satisfy do |exp|
          exp.mantissa == 1 && exp.exponent == 0
        end
      end
    end

    context 'status buy' do
      let(:initial_isu) { '10000000' }
      let(:x) do
        described_class::MItem.new(
          item_id: 1,
          power1: 1,
          power2: 1,
          power3: 3,
          power4: 2,
          price1: 1,
          price2: 1,
          price3: 7,
          price4: 6
        )
      end
      let(:y) do
        described_class::MItem.new(
          item_id: 2,
          power1: 1,
          power2: 1,
          power3: 7,
          power4: 6,
          price1: 1,
          price2: 1,
          price3: 3,
          price4: 2
        )
      end
      let(:mitems) do
        {
          1 => x,
          2 => y,
        }
      end
      let(:addings) do
        [
          described_class::Adding.new(nil, 0, initial_isu),
        ]
      end
      let(:buyings) do
        [
          described_class::Buying.new(nil, 1, 1, 100),
          described_class::Buying.new(nil, 1, 2, 200),
          described_class::Buying.new(nil, 2, 1, 300),
          described_class::Buying.new(nil, 2, 2, 2001),
        ]
      end

      it do
        expect(subject.adding.length).to eq(0)
        expect(subject.schedule.length).to eq(4)
        expect(subject.on_sale.length).to eq(2)
        expect(subject.items.length).to eq(2)
      end

      it do
        total_power = 0
        milli_isu = described_class.str2big(initial_isu) * 1000
        milli_isu -= x.get_price(1) * 1000
        milli_isu -= x.get_price(2) * 1000
        milli_isu -= y.get_price(1) * 1000
        milli_isu -= y.get_price(2) * 1000

        # 0sec
        expect(subject.schedule[0].time).to eq(0)
        expect(subject.schedule[0].milli_isu).to satisfy do |exp|
          exp2 = described_class.big2exp(milli_isu)
          exp.mantissa == exp2.mantissa && exp.exponent == exp2.exponent
        end
        expect(subject.schedule[0].total_power).to satisfy do |exp|
          exp2 = described_class.big2exp(total_power)
          exp.mantissa == exp2.mantissa && exp.exponent == exp2.exponent
        end

        # 0.1sec
        total_power += x.get_power(1)
        expect(subject.schedule[1].time).to eq(100)
        expect(subject.schedule[1].milli_isu).to satisfy do |exp|
          exp2 = described_class.big2exp(milli_isu)
          exp.mantissa == exp2.mantissa && exp.exponent == exp2.exponent
        end
        expect(subject.schedule[1].total_power).to satisfy do |exp|
          exp2 = described_class.big2exp(total_power)
          exp.mantissa == exp2.mantissa && exp.exponent == exp2.exponent
        end

        # 0.2sec
        milli_isu += total_power * 100
        total_power += x.get_power(2)
        expect(subject.schedule[2].time).to eq(200)
        expect(subject.schedule[2].milli_isu).to satisfy do |exp|
          exp2 = described_class.big2exp(milli_isu)
          exp.mantissa == exp2.mantissa && exp.exponent == exp2.exponent
        end
        expect(subject.schedule[2].total_power).to satisfy do |exp|
          exp2 = described_class.big2exp(total_power)
          exp.mantissa == exp2.mantissa && exp.exponent == exp2.exponent
        end

        # 0.3sec
        milli_isu += total_power * 100
        total_power += y.get_power(1)
        expect(subject.schedule[3].time).to eq(300)
        expect(subject.schedule[3].milli_isu).to satisfy do |exp|
          exp2 = described_class.big2exp(milli_isu)
          exp.mantissa == exp2.mantissa && exp.exponent == exp2.exponent
        end
        expect(subject.schedule[3].total_power).to satisfy do |exp|
          exp2 = described_class.big2exp(total_power)
          exp.mantissa == exp2.mantissa && exp.exponent == exp2.exponent
        end
      end

      it do
        expect(subject.on_sale).to satisfy do |on_sale|
          on_sale.any? { |o| o.item_id == 1 && o.time == 0 } && on_sale.any? { |o| o.item_id == 2 && o.time == 0 }
        end
      end
    end
  end

  describe '.big2exp' do
    let(:n) { described_class.str2big(s) }

    subject { described_class.big2exp(n) }

    context do
      let(:s) { '0' }

      it do
        is_expected.to satisfy do |exp|
          exp.mantissa == 0 && exp.exponent == 0
        end
      end
    end

    context do
      let(:s) { '1234' }

      it do
        is_expected.to satisfy do |exp|
          exp.mantissa == 1234 && exp.exponent == 0
        end
      end
    end

    context do
      let(:s) { '11111111111111000000' }

      it do
        is_expected.to satisfy do |exp|
          exp.mantissa == 111111111111110 && exp.exponent == 5
        end
      end
    end
  end
end

RSpec.describe Game::MItem do
  let(:item) do
    described_class.new(
      item_id: 1,
      power1: 1,
      power2: 2,
      power3: 2,
      power4: 3,
      price1: 5,
      price2: 4,
      price3: 3,
      price4: 2
    )
  end

  describe '#get_power' do
    let(:count) { 1 }

    subject { item.get_power(count) }

    it do
      is_expected.to eq(81)
    end
  end

  describe '#get_price' do
    let(:count) { 1 }

    subject { item.get_price(count) }

    it do
      is_expected.to eq(2048)
    end
  end
end
