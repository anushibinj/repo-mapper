import { useBilling } from '../hooks/useBilling';
import { InvoiceList } from '../components/InvoiceList';

export function BillingPage() {
  const { invoices, loading } = useBilling();

  if (loading) {
    return <div>Loading...</div>;
  }

  return (
    <div>
      <h1>Billing</h1>
      <InvoiceList invoices={invoices} />
    </div>
  );
}
