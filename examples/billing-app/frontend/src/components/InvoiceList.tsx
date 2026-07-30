export function InvoiceList({ invoices }) {
  return (
    <ul>
      {invoices.map((invoice) => (
        <li key={invoice.id}>{invoice.customerName} - {invoice.amount}</li>
      ))}
    </ul>
  );
}
