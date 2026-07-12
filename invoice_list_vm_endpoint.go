package main

import financevm "ph_holdings_app/internal/viewmodel/finance"

// GetInvoiceListVM returns a display-ready invoice list ViewModel.
func (a *App) GetInvoiceListVM(page, pageSize int) (financevm.InvoiceListVM, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize
	invoices, err := a.ListCustomerInvoices(pageSize, offset)
	if err != nil {
		return financevm.InvoiceListVM{}, err
	}
	return financevm.BuildInvoiceListVM(invoices, page, pageSize), nil
}
