import { useState, useEffect } from 'react';
import axios from 'axios';

export function useBilling() {
  const [invoices, setInvoices] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    axios.get('/billing/invoices').then((res) => {
      setInvoices(res.data);
      setLoading(false);
    });
  }, []);

  return { invoices, loading };
}
