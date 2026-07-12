import { writable } from 'svelte/store';

// Pending DN creation - set by OrdersScreen, consumed by DeliveryNotesScreen
interface PendingDNCreate {
    orderId: string;
    orderNumber: string;
    customerName: string;
    timestamp: number;
}

function createPendingDNStore() {
    const { subscribe, set } = writable<PendingDNCreate | null>(null);

    return {
        subscribe,
        request: (orderId: string, orderNumber: string, customerName: string) => {
            set({ orderId, orderNumber, customerName, timestamp: Date.now() });
        },
        clear: () => set(null)
    };
}

export const pendingDNCreate = createPendingDNStore();

// Pending project handoff (Wave 9.4 B4) - set by OpportunitiesScreen / OrdersScreen
// "Start project" actions, consumed by WorkHub on mount to preseed + open the
// project create composer with lineage (opportunity/order/customer/POC) already
// attached. Mirrors the pendingDNCreate pattern above.
export interface PendingProjectHandoff {
    source: 'opportunity' | 'order';
    sourceId: string;
    opportunityId?: string;
    orderId?: string;
    customerId?: string;
    customerName?: string;
    pocName?: string;
    pocEmail?: string;
    pocPhone?: string;
    suggestedName: string;
    timestamp: number;
}

function createPendingProjectHandoffStore() {
    const { subscribe, set } = writable<PendingProjectHandoff | null>(null);

    return {
        subscribe,
        request: (payload: Omit<PendingProjectHandoff, 'timestamp'>) => {
            set({ ...payload, timestamp: Date.now() });
        },
        clear: () => set(null),
    };
}

export const pendingProjectHandoff = createPendingProjectHandoffStore();

// Pending order view (Wave 9.5 B1b) - set by OffersScreen after MarkOfferWon,
// consumed by OperationsHub (switch to the Orders tab) + OrdersScreen (open the
// created order's detail). Gives the "View Order →" handoff a real destination
// instead of a toast-only dead end. Mirrors the pendingDNCreate pattern above.
interface PendingOrderView {
    orderId: string;
    orderNumber: string;
    timestamp: number;
}

function createPendingOrderViewStore() {
    const { subscribe, set } = writable<PendingOrderView | null>(null);

    return {
        subscribe,
        request: (orderId: string, orderNumber: string) => {
            set({ orderId, orderNumber, timestamp: Date.now() });
        },
        clear: () => set(null),
    };
}

export const pendingOrderView = createPendingOrderViewStore();

// Pending invoice creation (Wave 9.5 B7b) - set by DeliveryNotesScreen when a
// confirmed delivery brings an order's remaining-to-deliver to zero, consumed by
// InvoicesScreen to pre-open the create-invoice flow for that order (the sales
// loop's last handoff). Mirrors the pendingDNCreate pattern above.
interface PendingInvoiceCreate {
    orderId: string;
    orderNumber: string;
    customerName: string;
    timestamp: number;
}

function createPendingInvoiceCreateStore() {
    const { subscribe, set } = writable<PendingInvoiceCreate | null>(null);

    return {
        subscribe,
        request: (orderId: string, orderNumber: string, customerName: string) => {
            set({ orderId, orderNumber, customerName, timestamp: Date.now() });
        },
        clear: () => set(null),
    };
}

export const pendingInvoiceCreate = createPendingInvoiceCreateStore();
