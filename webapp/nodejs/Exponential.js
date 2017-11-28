class Exponential {
  constructor({ mantissa, exponent }) {
    this.mantissa = mantissa
    this.exponent = exponent
  }

  toJSON () {
    return [
      this.mantissa,
      this.exponent,
    ]
  }
}

module.exports = Exponential
