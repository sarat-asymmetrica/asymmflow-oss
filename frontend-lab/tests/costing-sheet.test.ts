/* Costing Sheet sacred-math tests — verbatim port of the old
 * CostingSheetScreen.svelte's calculateLineItem waterfall and its sheet-level
 * $derived totals. These pin the rounding order (Math.ceil only, at step 14)
 * and the specific fallback asymmetry (customs/handling/finance default
 * non-zero at calc-time; freight/margin default to 0 at calc-time — the "9"
 * and "20" are new-row seed values, never a calc-time rescue) so a future
 * edit can't silently "fix" them. */
import { describe, it, expect } from 'vitest'
import {
  calcLine,
  costingExportLine,
  createBlankLine,
  isValidLine,
  parseSeedLineItems,
  findMatchingCustomer,
  sheetTotals,
} from '../src/screens/costing-sheet-vm.svelte'
import type { CostingLineRow, CostingCustomerRow } from '../src/bridge/costing-sheet'

function line(overrides: Partial<CostingLineRow> = {}): CostingLineRow {
  return { ...createBlankLine(20), ...overrides }
}

describe('calcLine — sacred waterfall', () => {
  it('computes the full 16-step waterfall for a plain BHD line', () => {
    const l = line({
      currency: 'BHD',
      quantity: 2,
      fobForeign: 1000,
      freightPercent: 10,
      customsPercent: 5,
      handlingPercent: 4,
      financePercent: 1,
      insurance: 20,
      otherCosts: 5,
      marginPercent: 20,
    })
    const c = calcLine(l)
    expect(c.exchangeRate).toBe(1.0)
    expect(c.fobBHD).toBe(1000)
    expect(c.freightForeign).toBe(100)
    expect(c.freightBHD).toBe(100)
    expect(c.cf).toBe(1100)
    expect(c.customsBHD).toBeCloseTo(55, 6)
    expect(c.landedCost).toBeCloseTo(1175, 6)
    expect(c.handlingBHD).toBeCloseTo(47, 6)
    expect(c.financeBHD).toBeCloseTo(11.75, 6)
    expect(c.totalCost).toBeCloseTo(1238.75, 6)
    expect(c.sellingPrice).toBeCloseTo(1486.5, 6)
    // step 14: Math.ceil is the ONLY rounding.
    expect(c.suggestedPriceUnit).toBe(Math.ceil(1486.5))
    expect(c.effectivePrice).toBe(c.suggestedPriceUnit)
    expect(c.totalSuggestedPrice).toBe(c.suggestedPriceUnit * 2)
  })

  it('applies a non-BHD currency rate from the hardcoded FX table', () => {
    const c = calcLine(line({ currency: 'EUR', fobForeign: 100 }))
    expect(c.exchangeRate).toBe(0.45)
    expect(c.fobBHD).toBeCloseTo(45, 6)
  })

  it('falls back to 0.45 for an unrecognized currency', () => {
    const c = calcLine(line({ currency: 'XYZ', fobForeign: 100 }))
    expect(c.exchangeRate).toBe(0.45)
  })

  it('clamps quantity to at least 1 for qty=0', () => {
    const c = calcLine(line({ quantity: 0, fobForeign: 100 }))
    expect(c.quantity).toBe(1)
  })

  it('clamps a negative fobForeign to 0', () => {
    const c = calcLine(line({ fobForeign: -500 }))
    expect(c.fobBHD).toBe(0)
    expect(c.cf).toBe(0)
  })

  it('freight% falls back to 0 (not 9) at calc-time when invalid/blank', () => {
    // createBlankLine seeds freightPercent=9, but calc-time itself has NO
    // rescue default — verbatim nonNegativeNumber(freightPercent) with no
    // 2nd arg. Passing NaN through the row proves the calc-time behavior.
    const c = calcLine(line({ fobForeign: 1000, freightPercent: Number.NaN }))
    expect(c.freightForeign).toBe(0)
    expect(c.freightBHD).toBe(0)
  })

  it('margin% falls back to 0 (not 20) at calc-time when invalid/blank', () => {
    const c = calcLine(line({ fobForeign: 1000, marginPercent: Number.NaN }))
    expect(c.sellingPrice).toBe(c.totalCost) // 0% markup -> sellingPrice === totalCost
    expect(c.marginBHD).toBe(0)
  })

  it('customs/handling/finance default to 5/4/1 at calc-time when invalid', () => {
    // freightPercent pinned to 0 to isolate customs/handling/finance from the
    // freight-defaults-to-0 behavior covered by the test above.
    const c = calcLine(
      line({
        fobForeign: 1000,
        freightPercent: 0,
        customsPercent: Number.NaN,
        handlingPercent: Number.NaN,
        financePercent: Number.NaN,
      }),
    )
    expect(c.customsBHD).toBeCloseTo(1000 * 0.05, 6)
    expect(c.landedCost).toBeCloseTo(1050, 6)
    expect(c.handlingBHD).toBeCloseTo(1050 * 0.04, 6)
    expect(c.financeBHD).toBeCloseTo(1050 * 0.01, 6)
  })

  it('userPriceSet with userPrice=0 falls through to the suggested price', () => {
    const c = calcLine(line({ fobForeign: 1000, userPriceSet: true, userPrice: 0 }))
    expect(c.effectivePrice).toBe(c.suggestedPriceUnit)
  })

  it('userPriceSet with a positive userPrice overrides the suggested price', () => {
    const c = calcLine(line({ fobForeign: 1000, userPriceSet: true, userPrice: 42 }))
    expect(c.effectivePrice).toBe(42)
    expect(c.totalSuggestedPrice).toBe(42 * c.quantity)
  })

  it('a margin of 500% still rounds up via Math.ceil, never Math.round', () => {
    const c = calcLine(line({ fobForeign: 999_999_999, marginPercent: 500 }))
    expect(c.suggestedPriceUnit).toBe(Math.ceil(c.sellingPrice))
    expect(Number.isInteger(c.suggestedPriceUnit)).toBe(true)
  })
})

describe('isValidLine', () => {
  it('excludes an empty-equipment, zero-fob line', () => {
    expect(isValidLine(line({ equipment: '', fobForeign: 0 }))).toBe(false)
  })
  it('includes a line with only an equipment name', () => {
    expect(isValidLine(line({ equipment: 'Flow Meter', fobForeign: 0 }))).toBe(true)
  })
  it('includes a line with only a positive fob', () => {
    expect(isValidLine(line({ equipment: '', fobForeign: 1 }))).toBe(true)
  })
})

describe('sheet totals — verbatim asymmetry', () => {
  it('adds hiddenCharges to cost only — never touches revenue', () => {
    const lines = [line({ fobForeign: 1000 })]
    const withoutHidden = sheetTotals(lines, 0, 0, 10)
    const withHidden = sheetTotals(lines, 0, 500, 10)
    expect(withHidden.netAmount).toBe(withoutHidden.netAmount)
    expect(withHidden.grandTotal).toBe(withoutHidden.grandTotal)
    expect(withHidden.totalCost).toBeCloseTo(withoutHidden.totalCost + 500, 6)
  })

  it('applies discount and VAT to revenue only — never touches cost', () => {
    const lines = [line({ fobForeign: 1000 })]
    const base = sheetTotals(lines, 0, 0, 10)
    const discounted = sheetTotals(lines, 100, 0, 10)
    expect(discounted.totalCost).toBeCloseTo(base.totalCost, 6)
    expect(discounted.netAmount).toBeCloseTo(base.netAmount - 100, 6)
  })

  it('clamps VAT rate to 100 max', () => {
    const t = sheetTotals([line({ fobForeign: 100 })], 0, 0, 250)
    expect(t.vat).toBeCloseTo(t.netAmount, 6)
  })

  it('profitPercent is 0 when netAmount is 0 (never divides by zero)', () => {
    const t = sheetTotals([line({ equipment: '', fobForeign: 0 })], 0, 0, 10)
    expect(t.profitPercent).toBe(0)
  })
})

describe('parseSeedLineItems', () => {
  it('maps description/part_number/unit_price/quantity seed shape', () => {
    const json = JSON.stringify([{ description: 'Flow Meter', part_number: 'PN-1', unit_price: 250, quantity: 3, currency: 'EUR' }])
    const rows = parseSeedLineItems(json, 20)
    expect(rows).toHaveLength(1)
    expect(rows[0]!.equipment).toBe('Flow Meter')
    expect(rows[0]!.fobForeign).toBe(250)
    expect(rows[0]!.quantity).toBe(3)
    expect(rows[0]!.currency).toBe('EUR')
  })

  it('returns [] for malformed JSON rather than throwing', () => {
    expect(parseSeedLineItems('{not valid json')).toEqual([])
  })

  it('returns [] for an empty string', () => {
    expect(parseSeedLineItems('')).toEqual([])
  })

  it('falls back to BHD for an unrecognized seed currency', () => {
    const json = JSON.stringify([{ description: 'X', unit_price: 1, currency: 'ZZZ' }])
    expect(parseSeedLineItems(json)[0]!.currency).toBe('BHD')
  })
})

describe('findMatchingCustomer', () => {
  const customers: CostingCustomerRow[] = [
    { id: 'c1', businessName: 'Gulf Fabrication W.L.L.', contactPerson: 'A' },
    { id: 'c2', businessName: 'Manama Process Systems', contactPerson: 'B' },
  ]

  it('matches a near-duplicate name ignoring legal suffix punctuation', () => {
    const match = findMatchingCustomer(customers, 'Gulf Fabrication WLL')
    expect(match?.id).toBe('c1')
  })

  it('returns null for an empty name', () => {
    expect(findMatchingCustomer(customers, '')).toBeNull()
  })

  it('returns null when nothing matches', () => {
    expect(findMatchingCustomer(customers, 'Totally Unrelated Entity')).toBeNull()
  })
})

describe('costingExportLine — flat CostingExportData line mapping (R1.1)', () => {
  it("maps calcLine's computed outputs into the SaveCostingAsOffer line item", () => {
    const l = line({
      equipment: 'Coriolis Flow Meter',
      model: 'CFM-2200',
      currency: 'BHD',
      quantity: 2,
      fobForeign: 1000,
      freightPercent: 10,
      customsPercent: 5,
      handlingPercent: 4,
      financePercent: 1,
      insurance: 20,
      otherCosts: 5,
      marginPercent: 20,
    })
    const c = calcLine(l)
    const item = costingExportLine(l, 0)

    // slNo is 1-based; the offer-building fields come straight from the waterfall.
    expect(item.slNo).toBe(1)
    expect(item.equipment).toBe('Coriolis Flow Meter')
    expect(item.quantity).toBe(c.quantity)
    // suggestedPrice/totalPrice are what the backend uses to build offer items.
    expect(item.suggestedPrice).toBe(c.effectivePrice)
    expect(item.totalPrice).toBe(c.totalSuggestedPrice)
    // markupPercent is 0 so the offer item inherits the line's marginPercent.
    expect(item.markupPercent).toBe(0)
    expect(item.marginPercent).toBe(20)
    // detailed-costing persistence fields mirror calcLine.
    expect(item.fobBHD).toBe(c.fobBHD)
    expect(item.freightBHD).toBe(c.freightBHD)
    expect(item.customsBHD).toBe(c.customsBHD)
    expect(item.handlingBHD).toBe(c.handlingBHD)
    expect(item.financeBHD).toBe(c.financeBHD)
    expect(item.totalCost).toBe(c.totalCost)
    expect(item.exchangeRate).toBe(c.exchangeRate)
  })

  it('honors a user price override in suggestedPrice/totalPrice', () => {
    const l = line({ equipment: 'X', fobForeign: 500, quantity: 3, userPriceSet: true, userPrice: 999 })
    const item = costingExportLine(l, 4)
    expect(item.slNo).toBe(5)
    expect(item.suggestedPrice).toBe(999) // effectivePrice = user override
    expect(item.totalPrice).toBe(999 * 3)
    expect(item.userPriceSet).toBe(true)
    expect(item.userPrice).toBe(999)
  })
})
