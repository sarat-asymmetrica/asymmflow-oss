export namespace assets {
	
	export class Info {
	    id: string;
	    name: string;
	    description: string;
	    mime_type: string;
	    size: number;
	    created_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.mime_type = source["mime_type"];
	        this.size = source["size"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace banking {
	
	export class AllocationInput {
	    allocation_type: string;
	    entity_id: string;
	    allocated_amount: number;
	
	    static createFrom(source: any = {}) {
	        return new AllocationInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.allocation_type = source["allocation_type"];
	        this.entity_id = source["entity_id"];
	        this.allocated_amount = source["allocated_amount"];
	    }
	}
	export class BankReconciliationMatchResult {
	    matched_count: number;
	    unmatched_count: number;
	    total_lines: number;
	    matched_percent: number;
	    auto_matched_count: number;
	
	    static createFrom(source: any = {}) {
	        return new BankReconciliationMatchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.matched_count = source["matched_count"];
	        this.unmatched_count = source["unmatched_count"];
	        this.total_lines = source["total_lines"];
	        this.matched_percent = source["matched_percent"];
	        this.auto_matched_count = source["auto_matched_count"];
	    }
	}

}

export namespace butler {
	
	export class ButlerAction {
	    type: string;
	    target: string;
	    label: string;
	    data: any;
	
	    static createFrom(source: any = {}) {
	        return new ButlerAction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.target = source["target"];
	        this.label = source["label"];
	        this.data = source["data"];
	    }
	}
	export class ButlerResponseMetadata {
	    used_backend: string;
	    requested_model: string;
	    used_model: string;
	    fallback_reason: string;
	    finance_data_access: boolean;
	    context_mode: string;
	    data_coverage: string[];
	    entity_resolution: Record<string, any>;
	    generated_at: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new ButlerResponseMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.used_backend = source["used_backend"];
	        this.requested_model = source["requested_model"];
	        this.used_model = source["used_model"];
	        this.fallback_reason = source["fallback_reason"];
	        this.finance_data_access = source["finance_data_access"];
	        this.context_mode = source["context_mode"];
	        this.data_coverage = source["data_coverage"];
	        this.entity_resolution = source["entity_resolution"];
	        this.generated_at = source["generated_at"];
	        this.error = source["error"];
	    }
	}
	export class ButlerResponse {
	    message: string;
	    actions: ButlerAction[];
	    confidence: number;
	    context: Record<string, any>;
	    metadata: ButlerResponseMetadata;
	
	    static createFrom(source: any = {}) {
	        return new ButlerResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.actions = this.convertValues(source["actions"], ButlerAction);
	        this.confidence = source["confidence"];
	        this.context = source["context"];
	        this.metadata = this.convertValues(source["metadata"], ButlerResponseMetadata);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ChatMessage {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    conversation_id: string;
	    role: string;
	    content: string;
	    tokens_used: number;
	    message_type: string;
	    action_type: string;
	    action_target: string;
	    action_label: string;
	    action_data: string;
	    action_status: string;
	    action_metadata: string;
	
	    static createFrom(source: any = {}) {
	        return new ChatMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.conversation_id = source["conversation_id"];
	        this.role = source["role"];
	        this.content = source["content"];
	        this.tokens_used = source["tokens_used"];
	        this.message_type = source["message_type"];
	        this.action_type = source["action_type"];
	        this.action_target = source["action_target"];
	        this.action_label = source["action_label"];
	        this.action_data = source["action_data"];
	        this.action_status = source["action_status"];
	        this.action_metadata = source["action_metadata"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Conversation {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    title: string;
	    summary: string;
	    is_active: boolean;
	    last_msg_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new Conversation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.title = source["title"];
	        this.summary = source["summary"];
	        this.is_active = source["is_active"];
	        this.last_msg_at = this.convertValues(source["last_msg_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PredictionRecord {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    customer_id: string;
	    customer_name: string;
	    grade: string;
	    predicted_days: number;
	    confidence: number;
	    r1: number;
	    r2: number;
	    r3: number;
	
	    static createFrom(source: any = {}) {
	        return new PredictionRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.grade = source["grade"];
	        this.predicted_days = source["predicted_days"];
	        this.confidence = source["confidence"];
	        this.r1 = source["r1"];
	        this.r2 = source["r2"];
	        this.r3 = source["r3"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace cheque {
	
	export class OutstandingResult {
	    cheques: finance.OutstandingCheque[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new OutstandingResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cheques = this.convertValues(source["cheques"], finance.OutstandingCheque);
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace compliance {
	
	export class TaxRate {
	    name: string;
	    rate: number;
	    category: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new TaxRate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.rate = source["rate"];
	        this.category = source["category"];
	        this.description = source["description"];
	    }
	}
	export class ValidationEntry {
	    timestamp: time.Time;
	    event_name: string;
	    jurisdiction: string;
	    valid: boolean;
	    errors: string[];
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new ValidationEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = this.convertValues(source["timestamp"], time.Time);
	        this.event_name = source["event_name"];
	        this.jurisdiction = source["jurisdiction"];
	        this.valid = source["valid"];
	        this.errors = source["errors"];
	        this.warnings = source["warnings"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace contract {
	
	export class Contract {
	    id: string;
	    contract_no: string;
	    customer_id: string;
	    customer_name: string;
	    template_id?: string;
	    template_name: string;
	    contract_type: string;
	    contract_value_bhd: number;
	    payment_terms: string;
	    payment_grade: string;
	    advance_percent: number;
	    effective_date: time.Time;
	    expiry_date?: time.Time;
	    status: string;
	    pdf_path: string;
	    selected_clauses: string;
	    order_id?: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    created_by: string;
	    deleted_at?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new Contract(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.contract_no = source["contract_no"];
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.template_id = source["template_id"];
	        this.template_name = source["template_name"];
	        this.contract_type = source["contract_type"];
	        this.contract_value_bhd = source["contract_value_bhd"];
	        this.payment_terms = source["payment_terms"];
	        this.payment_grade = source["payment_grade"];
	        this.advance_percent = source["advance_percent"];
	        this.effective_date = this.convertValues(source["effective_date"], time.Time);
	        this.expiry_date = this.convertValues(source["expiry_date"], time.Time);
	        this.status = source["status"];
	        this.pdf_path = source["pdf_path"];
	        this.selected_clauses = source["selected_clauses"];
	        this.order_id = source["order_id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.created_by = source["created_by"];
	        this.deleted_at = this.convertValues(source["deleted_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Template {
	    id: string;
	    name: string;
	    category: string;
	    description: string;
	    content: string;
	    is_active: boolean;
	    created_at: time.Time;
	    updated_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new Template(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.category = source["category"];
	        this.description = source["description"];
	        this.content = source["content"];
	        this.is_active = source["is_active"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace crm {
	
	export class CustomerContact {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    customer_id: string;
	    contact_name: string;
	    job_title: string;
	    email: string;
	    phone: string;
	    address: string;
	    is_primary_contact: boolean;
	    is_primary: boolean;
	    salutation: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomerContact(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.customer_id = source["customer_id"];
	        this.contact_name = source["contact_name"];
	        this.job_title = source["job_title"];
	        this.email = source["email"];
	        this.phone = source["phone"];
	        this.address = source["address"];
	        this.is_primary_contact = source["is_primary_contact"];
	        this.is_primary = source["is_primary"];
	        this.salutation = source["salutation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CustomerMaster {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    customer_id: string;
	    customer_code: string;
	    customer_type: string;
	    business_name: string;
	    short_code: string;
	    trading_name: string;
	    cr_number: string;
	    status: string;
	    primary_phone: string;
	    primary_email: string;
	    website: string;
	    address_line1: string;
	    city: string;
	    country: string;
	    trn: string;
	    mobile_number: string;
	    tax_code: string;
	    address: string;
	    phone: string;
	    email: string;
	    industry: string;
	    relation_years: number;
	    payment_grade: string;
	    customer_grade: string;
	    payment_terms_days: number;
	    avg_payment_days: number;
	    dispute_count: number;
	    total_orders_value: number;
	    total_orders_count: number;
	    avg_order_value: number;
	    last_order_date?: time.Time;
	    ar_risk_tier: string;
	    outstanding_bhd: number;
	    overdue_days: number;
	    credit_limit_bhd: number;
	    is_credit_blocked: boolean;
	    requires_prepayment: boolean;
	    has_abb_competition: boolean;
	    is_emergency_only: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CustomerMaster(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.customer_id = source["customer_id"];
	        this.customer_code = source["customer_code"];
	        this.customer_type = source["customer_type"];
	        this.business_name = source["business_name"];
	        this.short_code = source["short_code"];
	        this.trading_name = source["trading_name"];
	        this.cr_number = source["cr_number"];
	        this.status = source["status"];
	        this.primary_phone = source["primary_phone"];
	        this.primary_email = source["primary_email"];
	        this.website = source["website"];
	        this.address_line1 = source["address_line1"];
	        this.city = source["city"];
	        this.country = source["country"];
	        this.trn = source["trn"];
	        this.mobile_number = source["mobile_number"];
	        this.tax_code = source["tax_code"];
	        this.address = source["address"];
	        this.phone = source["phone"];
	        this.email = source["email"];
	        this.industry = source["industry"];
	        this.relation_years = source["relation_years"];
	        this.payment_grade = source["payment_grade"];
	        this.customer_grade = source["customer_grade"];
	        this.payment_terms_days = source["payment_terms_days"];
	        this.avg_payment_days = source["avg_payment_days"];
	        this.dispute_count = source["dispute_count"];
	        this.total_orders_value = source["total_orders_value"];
	        this.total_orders_count = source["total_orders_count"];
	        this.avg_order_value = source["avg_order_value"];
	        this.last_order_date = this.convertValues(source["last_order_date"], time.Time);
	        this.ar_risk_tier = source["ar_risk_tier"];
	        this.outstanding_bhd = source["outstanding_bhd"];
	        this.overdue_days = source["overdue_days"];
	        this.credit_limit_bhd = source["credit_limit_bhd"];
	        this.is_credit_blocked = source["is_credit_blocked"];
	        this.requires_prepayment = source["requires_prepayment"];
	        this.has_abb_competition = source["has_abb_competition"];
	        this.is_emergency_only = source["is_emergency_only"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DBCostingAdditionalCost {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    costing_sheet_id: string;
	    description: string;
	    amount_bhd: number;
	
	    static createFrom(source: any = {}) {
	        return new DBCostingAdditionalCost(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.costing_sheet_id = source["costing_sheet_id"];
	        this.description = source["description"];
	        this.amount_bhd = source["amount_bhd"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DBCostingItem {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    costing_sheet_id: string;
	    line_number: number;
	    product_id: string;
	    product_type: string;
	    description: string;
	    quantity: number;
	    unit_cost_bhd: number;
	    margin_percent: number;
	    unit_price_bhd: number;
	    line_total_bhd: number;
	
	    static createFrom(source: any = {}) {
	        return new DBCostingItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.costing_sheet_id = source["costing_sheet_id"];
	        this.line_number = source["line_number"];
	        this.product_id = source["product_id"];
	        this.product_type = source["product_type"];
	        this.description = source["description"];
	        this.quantity = source["quantity"];
	        this.unit_cost_bhd = source["unit_cost_bhd"];
	        this.margin_percent = source["margin_percent"];
	        this.unit_price_bhd = source["unit_price_bhd"];
	        this.line_total_bhd = source["line_total_bhd"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DBCostingSheet {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    costing_number: string;
	    customer_id: string;
	    customer_name: string;
	    costing_date: time.Time;
	    valid_until: time.Time;
	    subtotal_bhd: number;
	    total_margin_bhd: number;
	    shipping_cost_bhd: number;
	    customs_duty_bhd: number;
	    clearance_cost_bhd: number;
	    handling_cost_bhd: number;
	    additional_costs_bhd: number;
	    grand_total_bhd: number;
	    status: string;
	    converted_to_offer_id: string;
	    items?: DBCostingItem[];
	    additional_costs?: DBCostingAdditionalCost[];
	
	    static createFrom(source: any = {}) {
	        return new DBCostingSheet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.costing_number = source["costing_number"];
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.costing_date = this.convertValues(source["costing_date"], time.Time);
	        this.valid_until = this.convertValues(source["valid_until"], time.Time);
	        this.subtotal_bhd = source["subtotal_bhd"];
	        this.total_margin_bhd = source["total_margin_bhd"];
	        this.shipping_cost_bhd = source["shipping_cost_bhd"];
	        this.customs_duty_bhd = source["customs_duty_bhd"];
	        this.clearance_cost_bhd = source["clearance_cost_bhd"];
	        this.handling_cost_bhd = source["handling_cost_bhd"];
	        this.additional_costs_bhd = source["additional_costs_bhd"];
	        this.grand_total_bhd = source["grand_total_bhd"];
	        this.status = source["status"];
	        this.converted_to_offer_id = source["converted_to_offer_id"];
	        this.items = this.convertValues(source["items"], DBCostingItem);
	        this.additional_costs = this.convertValues(source["additional_costs"], DBCostingAdditionalCost);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DeliveryNoteItem {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    delivery_note_id: string;
	    order_item_id: string;
	    product_id: string;
	    product_code: string;
	    description: string;
	    quantity_ordered: number;
	    quantity_delivered: number;
	    quantity_remaining: number;
	
	    static createFrom(source: any = {}) {
	        return new DeliveryNoteItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.delivery_note_id = source["delivery_note_id"];
	        this.order_item_id = source["order_item_id"];
	        this.product_id = source["product_id"];
	        this.product_code = source["product_code"];
	        this.description = source["description"];
	        this.quantity_ordered = source["quantity_ordered"];
	        this.quantity_delivered = source["quantity_delivered"];
	        this.quantity_remaining = source["quantity_remaining"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DeliveryNote {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    order_id: string;
	    customer_id: string;
	    dn_number: string;
	    delivery_date: time.Time;
	    delivery_address: string;
	    contact_person: string;
	    contact_phone: string;
	    driver_name: string;
	    vehicle_number: string;
	    transport_method: string;
	    status: string;
	    updated_by: string;
	    signed_by: string;
	    signed_at?: time.Time;
	    signature_image: string;
	    is_partial_delivery: boolean;
	    delivery_sequence: number;
	    total_deliveries: number;
	    items?: DeliveryNoteItem[];
	
	    static createFrom(source: any = {}) {
	        return new DeliveryNote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.order_id = source["order_id"];
	        this.customer_id = source["customer_id"];
	        this.dn_number = source["dn_number"];
	        this.delivery_date = this.convertValues(source["delivery_date"], time.Time);
	        this.delivery_address = source["delivery_address"];
	        this.contact_person = source["contact_person"];
	        this.contact_phone = source["contact_phone"];
	        this.driver_name = source["driver_name"];
	        this.vehicle_number = source["vehicle_number"];
	        this.transport_method = source["transport_method"];
	        this.status = source["status"];
	        this.updated_by = source["updated_by"];
	        this.signed_by = source["signed_by"];
	        this.signed_at = this.convertValues(source["signed_at"], time.Time);
	        this.signature_image = source["signature_image"];
	        this.is_partial_delivery = source["is_partial_delivery"];
	        this.delivery_sequence = source["delivery_sequence"];
	        this.total_deliveries = source["total_deliveries"];
	        this.items = this.convertValues(source["items"], DeliveryNoteItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class EntityNote {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    entity_type: string;
	    entity_id: string;
	    note_type: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new EntityNote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.entity_type = source["entity_type"];
	        this.entity_id = source["entity_id"];
	        this.note_type = source["note_type"];
	        this.content = source["content"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FollowUpTask {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    customer_id: string;
	    title: string;
	    description: string;
	    due_date: time.Time;
	    status: string;
	    priority: string;
	    type: string;
	    amount: number;
	    contact: string;
	    notes: string;
	    completed_at?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new FollowUpTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.customer_id = source["customer_id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.due_date = this.convertValues(source["due_date"], time.Time);
	        this.status = source["status"];
	        this.priority = source["priority"];
	        this.type = source["type"];
	        this.amount = source["amount"];
	        this.contact = source["contact"];
	        this.notes = source["notes"];
	        this.completed_at = this.convertValues(source["completed_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GRNItem {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    grn_id: string;
	    po_item_id: string;
	    product_id: string;
	    quantity_ordered: number;
	    quantity_received: number;
	    quantity_accepted: number;
	    quantity_rejected: number;
	    rejection_reason: string;
	
	    static createFrom(source: any = {}) {
	        return new GRNItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.grn_id = source["grn_id"];
	        this.po_item_id = source["po_item_id"];
	        this.product_id = source["product_id"];
	        this.quantity_ordered = source["quantity_ordered"];
	        this.quantity_received = source["quantity_received"];
	        this.quantity_accepted = source["quantity_accepted"];
	        this.quantity_rejected = source["quantity_rejected"];
	        this.rejection_reason = source["rejection_reason"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GoodsReceivedNote {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    purchase_order_id: string;
	    grn_number: string;
	    received_date: time.Time;
	    received_by: string;
	    warehouse_id: string;
	    supplier_dn_number: string;
	    qc_status: string;
	    qc_notes: string;
	    qc_date?: time.Time;
	    qc_by: string;
	    completed_at?: time.Time;
	    updated_by: string;
	    items?: GRNItem[];
	
	    static createFrom(source: any = {}) {
	        return new GoodsReceivedNote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.purchase_order_id = source["purchase_order_id"];
	        this.grn_number = source["grn_number"];
	        this.received_date = this.convertValues(source["received_date"], time.Time);
	        this.received_by = source["received_by"];
	        this.warehouse_id = source["warehouse_id"];
	        this.supplier_dn_number = source["supplier_dn_number"];
	        this.qc_status = source["qc_status"];
	        this.qc_notes = source["qc_notes"];
	        this.qc_date = this.convertValues(source["qc_date"], time.Time);
	        this.qc_by = source["qc_by"];
	        this.completed_at = this.convertValues(source["completed_at"], time.Time);
	        this.updated_by = source["updated_by"];
	        this.items = this.convertValues(source["items"], GRNItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InventoryItem {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    product_id: string;
	    product_code: string;
	    warehouse_id: string;
	    quantity_on_hand: number;
	    quantity_reserved: number;
	    quantity_available: number;
	    unit_cost: number;
	    stock_status: string;
	    is_active: boolean;
	    reorder_point: number;
	    minimum_stock: number;
	    maximum_stock: number;
	    total_value: number;
	    last_purchase_cost: number;
	    last_movement_at?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new InventoryItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.product_id = source["product_id"];
	        this.product_code = source["product_code"];
	        this.warehouse_id = source["warehouse_id"];
	        this.quantity_on_hand = source["quantity_on_hand"];
	        this.quantity_reserved = source["quantity_reserved"];
	        this.quantity_available = source["quantity_available"];
	        this.unit_cost = source["unit_cost"];
	        this.stock_status = source["stock_status"];
	        this.is_active = source["is_active"];
	        this.reorder_point = source["reorder_point"];
	        this.minimum_stock = source["minimum_stock"];
	        this.maximum_stock = source["maximum_stock"];
	        this.total_value = source["total_value"];
	        this.last_purchase_cost = source["last_purchase_cost"];
	        this.last_movement_at = this.convertValues(source["last_movement_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OfferItem {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    offer_id: string;
	    line_number: number;
	    product_id: string;
	    product_code: string;
	    model: string;
	    description: string;
	    quantity: number;
	    unit_price_bhd: number;
	    long_code: string;
	    equipment: string;
	    specification: string;
	    detailed_description: string;
	    currency: string;
	    fob: number;
	    freight: number;
	    total_cost: number;
	    margin_percent: number;
	    total_price: number;
	    exchange_rate: number;
	    fob_bhd: number;
	    freight_bhd: number;
	    insurance: number;
	    customs_percent: number;
	    customs_bhd: number;
	    handling_percent: number;
	    handling_bhd: number;
	    finance_percent: number;
	    finance_bhd: number;
	    other_costs: number;
	    user_price: number;
	    user_price_set: boolean;
	
	    static createFrom(source: any = {}) {
	        return new OfferItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.offer_id = source["offer_id"];
	        this.line_number = source["line_number"];
	        this.product_id = source["product_id"];
	        this.product_code = source["product_code"];
	        this.model = source["model"];
	        this.description = source["description"];
	        this.quantity = source["quantity"];
	        this.unit_price_bhd = source["unit_price_bhd"];
	        this.long_code = source["long_code"];
	        this.equipment = source["equipment"];
	        this.specification = source["specification"];
	        this.detailed_description = source["detailed_description"];
	        this.currency = source["currency"];
	        this.fob = source["fob"];
	        this.freight = source["freight"];
	        this.total_cost = source["total_cost"];
	        this.margin_percent = source["margin_percent"];
	        this.total_price = source["total_price"];
	        this.exchange_rate = source["exchange_rate"];
	        this.fob_bhd = source["fob_bhd"];
	        this.freight_bhd = source["freight_bhd"];
	        this.insurance = source["insurance"];
	        this.customs_percent = source["customs_percent"];
	        this.customs_bhd = source["customs_bhd"];
	        this.handling_percent = source["handling_percent"];
	        this.handling_bhd = source["handling_bhd"];
	        this.finance_percent = source["finance_percent"];
	        this.finance_bhd = source["finance_bhd"];
	        this.other_costs = source["other_costs"];
	        this.user_price = source["user_price"];
	        this.user_price_set = source["user_price_set"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Offer {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    offer_number: string;
	    revision_number: number;
	    revision_of_offer_id: string;
	    revision_root_offer_id: string;
	    superseded_by_offer_id: string;
	    superseded_at?: time.Time;
	    rfq_id: string;
	    customer_id: string;
	    customer_name: string;
	    quotation_date: time.Time;
	    validity_date: time.Time;
	    total_value_bhd: number;
	    estimated_margin: number;
	    stage: string;
	    has_abb_competition: boolean;
	    lost_reason: string;
	    payment_terms: string;
	    delivery_terms: string;
	    delivery_weeks: string;
	    country_of_origin: string;
	    issued_by: string;
	    contact_phone: string;
	    customer_reference: string;
	    attention_person: string;
	    attention_company: string;
	    attention_phone: string;
	    attention_address: string;
	    discount_percent: number;
	    quote_type: string;
	    vat_rate: number;
	    attachment_scope_id: string;
	    division: string;
	    terms_and_conditions: string;
	    subject: string;
	    body: string;
	    coc_coo: string;
	    test_certificate: string;
	    installation: string;
	    commissioning: string;
	    testing: string;
	    folder_number: string;
	    project_name: string;
	    items?: OfferItem[];
	
	    static createFrom(source: any = {}) {
	        return new Offer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.offer_number = source["offer_number"];
	        this.revision_number = source["revision_number"];
	        this.revision_of_offer_id = source["revision_of_offer_id"];
	        this.revision_root_offer_id = source["revision_root_offer_id"];
	        this.superseded_by_offer_id = source["superseded_by_offer_id"];
	        this.superseded_at = this.convertValues(source["superseded_at"], time.Time);
	        this.rfq_id = source["rfq_id"];
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.quotation_date = this.convertValues(source["quotation_date"], time.Time);
	        this.validity_date = this.convertValues(source["validity_date"], time.Time);
	        this.total_value_bhd = source["total_value_bhd"];
	        this.estimated_margin = source["estimated_margin"];
	        this.stage = source["stage"];
	        this.has_abb_competition = source["has_abb_competition"];
	        this.lost_reason = source["lost_reason"];
	        this.payment_terms = source["payment_terms"];
	        this.delivery_terms = source["delivery_terms"];
	        this.delivery_weeks = source["delivery_weeks"];
	        this.country_of_origin = source["country_of_origin"];
	        this.issued_by = source["issued_by"];
	        this.contact_phone = source["contact_phone"];
	        this.customer_reference = source["customer_reference"];
	        this.attention_person = source["attention_person"];
	        this.attention_company = source["attention_company"];
	        this.attention_phone = source["attention_phone"];
	        this.attention_address = source["attention_address"];
	        this.discount_percent = source["discount_percent"];
	        this.quote_type = source["quote_type"];
	        this.vat_rate = source["vat_rate"];
	        this.attachment_scope_id = source["attachment_scope_id"];
	        this.division = source["division"];
	        this.terms_and_conditions = source["terms_and_conditions"];
	        this.subject = source["subject"];
	        this.body = source["body"];
	        this.coc_coo = source["coc_coo"];
	        this.test_certificate = source["test_certificate"];
	        this.installation = source["installation"];
	        this.commissioning = source["commissioning"];
	        this.testing = source["testing"];
	        this.folder_number = source["folder_number"];
	        this.project_name = source["project_name"];
	        this.items = this.convertValues(source["items"], OfferItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OfferFollowUp {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    offer_id: string;
	    follow_up_date: time.Time;
	    notes: string;
	    status: string;
	    completed_at?: time.Time;
	    completed_by: string;
	
	    static createFrom(source: any = {}) {
	        return new OfferFollowUp(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.offer_id = source["offer_id"];
	        this.follow_up_date = this.convertValues(source["follow_up_date"], time.Time);
	        this.notes = source["notes"];
	        this.status = source["status"];
	        this.completed_at = this.convertValues(source["completed_at"], time.Time);
	        this.completed_by = source["completed_by"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class OfferNote {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    offer_id: string;
	    note_date: time.Time;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new OfferNote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.offer_id = source["offer_id"];
	        this.note_date = this.convertValues(source["note_date"], time.Time);
	        this.content = source["content"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Opportunity {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    folder_number: string;
	    offer_id: string;
	    customer_id: string;
	    customer_name: string;
	    customer_grade: string;
	    salesperson: string;
	    division: string;
	    year: number;
	    opp_number: number;
	    folder_name: string;
	    title: string;
	    eh_ref: string;
	    source: string;
	    comment: string;
	    owner_notes: string;
	    product_details: string;
	    offer_date: time.Time;
	    order_date?: time.Time;
	    expected_date?: time.Time;
	    closed_date?: time.Time;
	    delivery_terms: string;
	    payment_terms: string;
	    revenue_bhd: number;
	    cost_bhd: number;
	    profit_bhd: number;
	    spoc_status: string;
	    wip_status: string;
	    stage: string;
	    regime: number;
	    confidence: number;
	    r1: number;
	    r2: number;
	    r3: number;
	    has_abb_competition: boolean;
	    product_type: string;
	    won_reason: string;
	    lost_reason: string;
	
	    static createFrom(source: any = {}) {
	        return new Opportunity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.folder_number = source["folder_number"];
	        this.offer_id = source["offer_id"];
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.customer_grade = source["customer_grade"];
	        this.salesperson = source["salesperson"];
	        this.division = source["division"];
	        this.year = source["year"];
	        this.opp_number = source["opp_number"];
	        this.folder_name = source["folder_name"];
	        this.title = source["title"];
	        this.eh_ref = source["eh_ref"];
	        this.source = source["source"];
	        this.comment = source["comment"];
	        this.owner_notes = source["owner_notes"];
	        this.product_details = source["product_details"];
	        this.offer_date = this.convertValues(source["offer_date"], time.Time);
	        this.order_date = this.convertValues(source["order_date"], time.Time);
	        this.expected_date = this.convertValues(source["expected_date"], time.Time);
	        this.closed_date = this.convertValues(source["closed_date"], time.Time);
	        this.delivery_terms = source["delivery_terms"];
	        this.payment_terms = source["payment_terms"];
	        this.revenue_bhd = source["revenue_bhd"];
	        this.cost_bhd = source["cost_bhd"];
	        this.profit_bhd = source["profit_bhd"];
	        this.spoc_status = source["spoc_status"];
	        this.wip_status = source["wip_status"];
	        this.stage = source["stage"];
	        this.regime = source["regime"];
	        this.confidence = source["confidence"];
	        this.r1 = source["r1"];
	        this.r2 = source["r2"];
	        this.r3 = source["r3"];
	        this.has_abb_competition = source["has_abb_competition"];
	        this.product_type = source["product_type"];
	        this.won_reason = source["won_reason"];
	        this.lost_reason = source["lost_reason"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OrderItem {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    order_id: string;
	    line_number: number;
	    product_id: string;
	    product_code: string;
	    description: string;
	    quantity: number;
	    unit_price_bhd: number;
	    quantity_shipped: number;
	    quantity_invoiced: number;
	    equipment: string;
	    model: string;
	    specification: string;
	    detailed_description: string;
	    currency: string;
	    fob: number;
	    freight: number;
	    total_cost: number;
	    margin_percent: number;
	    total_price: number;
	    brand: string;
	    token: string;
	
	    static createFrom(source: any = {}) {
	        return new OrderItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.order_id = source["order_id"];
	        this.line_number = source["line_number"];
	        this.product_id = source["product_id"];
	        this.product_code = source["product_code"];
	        this.description = source["description"];
	        this.quantity = source["quantity"];
	        this.unit_price_bhd = source["unit_price_bhd"];
	        this.quantity_shipped = source["quantity_shipped"];
	        this.quantity_invoiced = source["quantity_invoiced"];
	        this.equipment = source["equipment"];
	        this.model = source["model"];
	        this.specification = source["specification"];
	        this.detailed_description = source["detailed_description"];
	        this.currency = source["currency"];
	        this.fob = source["fob"];
	        this.freight = source["freight"];
	        this.total_cost = source["total_cost"];
	        this.margin_percent = source["margin_percent"];
	        this.total_price = source["total_price"];
	        this.brand = source["brand"];
	        this.token = source["token"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Order {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    order_number: string;
	    customer_po_number: string;
	    customer_id: string;
	    customer_name: string;
	    order_date: time.Time;
	    required_date: time.Time;
	    total_value_bhd: number;
	    grand_total_bhd: number;
	    status: string;
	    updated_by: string;
	    payment_terms: string;
	    delivery_terms: string;
	    offer_id: string;
	    offer_number: string;
	    rfq_id: string;
	    customer_reference: string;
	    attention_person: string;
	    attention_company: string;
	    attention_phone: string;
	    attention_address: string;
	    delivery_weeks: string;
	    country_of_origin: string;
	    issued_by: string;
	    contact_phone: string;
	    discount_percent: number;
	    division: string;
	    items?: OrderItem[];
	
	    static createFrom(source: any = {}) {
	        return new Order(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.order_number = source["order_number"];
	        this.customer_po_number = source["customer_po_number"];
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.order_date = this.convertValues(source["order_date"], time.Time);
	        this.required_date = this.convertValues(source["required_date"], time.Time);
	        this.total_value_bhd = source["total_value_bhd"];
	        this.grand_total_bhd = source["grand_total_bhd"];
	        this.status = source["status"];
	        this.updated_by = source["updated_by"];
	        this.payment_terms = source["payment_terms"];
	        this.delivery_terms = source["delivery_terms"];
	        this.offer_id = source["offer_id"];
	        this.offer_number = source["offer_number"];
	        this.rfq_id = source["rfq_id"];
	        this.customer_reference = source["customer_reference"];
	        this.attention_person = source["attention_person"];
	        this.attention_company = source["attention_company"];
	        this.attention_phone = source["attention_phone"];
	        this.attention_address = source["attention_address"];
	        this.delivery_weeks = source["delivery_weeks"];
	        this.country_of_origin = source["country_of_origin"];
	        this.issued_by = source["issued_by"];
	        this.contact_phone = source["contact_phone"];
	        this.discount_percent = source["discount_percent"];
	        this.division = source["division"];
	        this.items = this.convertValues(source["items"], OrderItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class PostSaleNote {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    order_id: string;
	    order_number: string;
	    note_type: string;
	    description: string;
	    cost_bhd: number;
	    note_date: time.Time;
	    resolved_at?: time.Time;
	    resolution: string;
	
	    static createFrom(source: any = {}) {
	        return new PostSaleNote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.order_id = source["order_id"];
	        this.order_number = source["order_number"];
	        this.note_type = source["note_type"];
	        this.description = source["description"];
	        this.cost_bhd = source["cost_bhd"];
	        this.note_date = this.convertValues(source["note_date"], time.Time);
	        this.resolved_at = this.convertValues(source["resolved_at"], time.Time);
	        this.resolution = source["resolution"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProductMaster {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    product_code: string;
	    product_name: string;
	    product_category: string;
	    supplier_id: string;
	    supplier_code: string;
	    standard_cost_bhd: number;
	    standard_price_bhd: number;
	    description: string;
	    is_active: boolean;
	    stock_quantity: number;
	    sku: string;
	    part_number: string;
	    hs_code: string;
	    unit_of_measure: string;
	    datasheet_url: string;
	    specifications: string;
	    requires_serial_tracking: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProductMaster(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.product_code = source["product_code"];
	        this.product_name = source["product_name"];
	        this.product_category = source["product_category"];
	        this.supplier_id = source["supplier_id"];
	        this.supplier_code = source["supplier_code"];
	        this.standard_cost_bhd = source["standard_cost_bhd"];
	        this.standard_price_bhd = source["standard_price_bhd"];
	        this.description = source["description"];
	        this.is_active = source["is_active"];
	        this.stock_quantity = source["stock_quantity"];
	        this.sku = source["sku"];
	        this.part_number = source["part_number"];
	        this.hs_code = source["hs_code"];
	        this.unit_of_measure = source["unit_of_measure"];
	        this.datasheet_url = source["datasheet_url"];
	        this.specifications = source["specifications"];
	        this.requires_serial_tracking = source["requires_serial_tracking"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PurchaseOrderItem {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    purchase_order_id: string;
	    order_item_id: string;
	    product_id: string;
	    product_code: string;
	    supplier_part_number: string;
	    description: string;
	    quantity: number;
	    unit_price_foreign: number;
	    unit_price_bhd: number;
	    total_foreign: number;
	    total_bhd: number;
	    quantity_received: number;
	    requires_serial_tracking: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PurchaseOrderItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.purchase_order_id = source["purchase_order_id"];
	        this.order_item_id = source["order_item_id"];
	        this.product_id = source["product_id"];
	        this.product_code = source["product_code"];
	        this.supplier_part_number = source["supplier_part_number"];
	        this.description = source["description"];
	        this.quantity = source["quantity"];
	        this.unit_price_foreign = source["unit_price_foreign"];
	        this.unit_price_bhd = source["unit_price_bhd"];
	        this.total_foreign = source["total_foreign"];
	        this.total_bhd = source["total_bhd"];
	        this.quantity_received = source["quantity_received"];
	        this.requires_serial_tracking = source["requires_serial_tracking"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PurchaseOrder {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    order_id: string;
	    rfq_id: string;
	    supplier_id: string;
	    supplier_name: string;
	    po_number: string;
	    po_date: time.Time;
	    expected_delivery: time.Time;
	    currency: string;
	    exchange_rate: number;
	    subtotal_foreign: number;
	    subtotal_bhd: number;
	    vat_amount: number;
	    total_foreign: number;
	    total_bhd: number;
	    payment_terms: string;
	    payment_due_date: time.Time;
	    status: string;
	    approved_by: string;
	    approved_at?: time.Time;
	    updated_by: string;
	    division: string;
	    items?: PurchaseOrderItem[];
	
	    static createFrom(source: any = {}) {
	        return new PurchaseOrder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.order_id = source["order_id"];
	        this.rfq_id = source["rfq_id"];
	        this.supplier_id = source["supplier_id"];
	        this.supplier_name = source["supplier_name"];
	        this.po_number = source["po_number"];
	        this.po_date = this.convertValues(source["po_date"], time.Time);
	        this.expected_delivery = this.convertValues(source["expected_delivery"], time.Time);
	        this.currency = source["currency"];
	        this.exchange_rate = source["exchange_rate"];
	        this.subtotal_foreign = source["subtotal_foreign"];
	        this.subtotal_bhd = source["subtotal_bhd"];
	        this.vat_amount = source["vat_amount"];
	        this.total_foreign = source["total_foreign"];
	        this.total_bhd = source["total_bhd"];
	        this.payment_terms = source["payment_terms"];
	        this.payment_due_date = this.convertValues(source["payment_due_date"], time.Time);
	        this.status = source["status"];
	        this.approved_by = source["approved_by"];
	        this.approved_at = this.convertValues(source["approved_at"], time.Time);
	        this.updated_by = source["updated_by"];
	        this.division = source["division"];
	        this.items = this.convertValues(source["items"], PurchaseOrderItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SerialNumber {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    product_id: string;
	    product_code: string;
	    serial_no: string;
	    lot_number: string;
	    status: string;
	    po_id: string;
	    po_number: string;
	    grn_item_id: string;
	    grn_number: string;
	    dn_item_id: string;
	    dn_number: string;
	    invoice_id: string;
	    invoice_number: string;
	    customer_id: string;
	    customer_name: string;
	    received_date?: time.Time;
	    shipped_date?: time.Time;
	    warranty_start_date?: time.Time;
	    warranty_end_date?: time.Time;
	    warranty_months: number;
	    calibration_date?: time.Time;
	    calibration_due_date?: time.Time;
	    calibration_cert_path: string;
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new SerialNumber(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.product_id = source["product_id"];
	        this.product_code = source["product_code"];
	        this.serial_no = source["serial_no"];
	        this.lot_number = source["lot_number"];
	        this.status = source["status"];
	        this.po_id = source["po_id"];
	        this.po_number = source["po_number"];
	        this.grn_item_id = source["grn_item_id"];
	        this.grn_number = source["grn_number"];
	        this.dn_item_id = source["dn_item_id"];
	        this.dn_number = source["dn_number"];
	        this.invoice_id = source["invoice_id"];
	        this.invoice_number = source["invoice_number"];
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.received_date = this.convertValues(source["received_date"], time.Time);
	        this.shipped_date = this.convertValues(source["shipped_date"], time.Time);
	        this.warranty_start_date = this.convertValues(source["warranty_start_date"], time.Time);
	        this.warranty_end_date = this.convertValues(source["warranty_end_date"], time.Time);
	        this.warranty_months = source["warranty_months"];
	        this.calibration_date = this.convertValues(source["calibration_date"], time.Time);
	        this.calibration_due_date = this.convertValues(source["calibration_due_date"], time.Time);
	        this.calibration_cert_path = source["calibration_cert_path"];
	        this.notes = source["notes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class StockAdjustment {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    inventory_item_id: string;
	    adjustment_date: time.Time;
	    adjustment_type: string;
	    reason: string;
	    variance: number;
	    system_quantity: number;
	    physical_quantity: number;
	    unit_cost: number;
	    value_impact: number;
	    notes: string;
	    status: string;
	    adjustment_number: string;
	    approved_by: string;
	    approved_at?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new StockAdjustment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.inventory_item_id = source["inventory_item_id"];
	        this.adjustment_date = this.convertValues(source["adjustment_date"], time.Time);
	        this.adjustment_type = source["adjustment_type"];
	        this.reason = source["reason"];
	        this.variance = source["variance"];
	        this.system_quantity = source["system_quantity"];
	        this.physical_quantity = source["physical_quantity"];
	        this.unit_cost = source["unit_cost"];
	        this.value_impact = source["value_impact"];
	        this.notes = source["notes"];
	        this.status = source["status"];
	        this.adjustment_number = source["adjustment_number"];
	        this.approved_by = source["approved_by"];
	        this.approved_at = this.convertValues(source["approved_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class StockMovement {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    inventory_item_id: string;
	    movement_type: string;
	    movement_number: string;
	    quantity: number;
	    direction: string;
	    balance_before: number;
	    balance_after: number;
	    movement_date: time.Time;
	    reference_type: string;
	    reference_id: string;
	    reference_number: string;
	    counterparty_id: string;
	    counterparty_name: string;
	    notes: string;
	    unit_cost: number;
	    total_value: number;
	
	    static createFrom(source: any = {}) {
	        return new StockMovement(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.inventory_item_id = source["inventory_item_id"];
	        this.movement_type = source["movement_type"];
	        this.movement_number = source["movement_number"];
	        this.quantity = source["quantity"];
	        this.direction = source["direction"];
	        this.balance_before = source["balance_before"];
	        this.balance_after = source["balance_after"];
	        this.movement_date = this.convertValues(source["movement_date"], time.Time);
	        this.reference_type = source["reference_type"];
	        this.reference_id = source["reference_id"];
	        this.reference_number = source["reference_number"];
	        this.counterparty_id = source["counterparty_id"];
	        this.counterparty_name = source["counterparty_name"];
	        this.notes = source["notes"];
	        this.unit_cost = source["unit_cost"];
	        this.total_value = source["total_value"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SupplierContact {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    supplier_id: string;
	    contact_name: string;
	    job_title: string;
	    email: string;
	    phone: string;
	    address: string;
	    is_primary_contact: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SupplierContact(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.supplier_id = source["supplier_id"];
	        this.contact_name = source["contact_name"];
	        this.job_title = source["job_title"];
	        this.email = source["email"];
	        this.phone = source["phone"];
	        this.address = source["address"];
	        this.is_primary_contact = source["is_primary_contact"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SupplierIssue {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    supplier_id: string;
	    order_ref: string;
	    description: string;
	    status: string;
	    resolution: string;
	    cost_bhd: number;
	    resolved_at?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new SupplierIssue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.supplier_id = source["supplier_id"];
	        this.order_ref = source["order_ref"];
	        this.description = source["description"];
	        this.status = source["status"];
	        this.resolution = source["resolution"];
	        this.cost_bhd = source["cost_bhd"];
	        this.resolved_at = this.convertValues(source["resolved_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SupplierMaster {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    supplier_code: string;
	    supplier_name: string;
	    country: string;
	    lead_time_days: number;
	    tax_id: string;
	    supplier_type: string;
	    brands_handled: string;
	    product_types: string;
	    primary_contact: string;
	    email: string;
	    phone: string;
	    address: string;
	    bank_name: string;
	    account_number: string;
	    iban: string;
	    swift_code: string;
	    payment_terms: string;
	    rating: number;
	    notes: string;
	    is_active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SupplierMaster(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.supplier_code = source["supplier_code"];
	        this.supplier_name = source["supplier_name"];
	        this.country = source["country"];
	        this.lead_time_days = source["lead_time_days"];
	        this.tax_id = source["tax_id"];
	        this.supplier_type = source["supplier_type"];
	        this.brands_handled = source["brands_handled"];
	        this.product_types = source["product_types"];
	        this.primary_contact = source["primary_contact"];
	        this.email = source["email"];
	        this.phone = source["phone"];
	        this.address = source["address"];
	        this.bank_name = source["bank_name"];
	        this.account_number = source["account_number"];
	        this.iban = source["iban"];
	        this.swift_code = source["swift_code"];
	        this.payment_terms = source["payment_terms"];
	        this.rating = source["rating"];
	        this.notes = source["notes"];
	        this.is_active = source["is_active"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Warehouse {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    code: string;
	    name: string;
	    is_active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Warehouse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.code = source["code"];
	        this.name = source["name"];
	        this.is_active = source["is_active"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace data {
	
	export class ImportResult {
	    customers_total: number;
	    customers_imported: number;
	    customers_skipped: number;
	    customer_errors?: string[];
	    opportunities_total: number;
	    opportunities_imported: number;
	    opportunities_skipped: number;
	    opportunity_errors?: string[];
	    payments_total: number;
	    payments_imported: number;
	    payments_skipped: number;
	    payment_errors?: string[];
	    products_total: number;
	    products_imported: number;
	    products_skipped: number;
	    product_errors?: string[];
	    total_records: number;
	    total_imported: number;
	    duration: number;
	    start_time: time.Time;
	    end_time: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.customers_total = source["customers_total"];
	        this.customers_imported = source["customers_imported"];
	        this.customers_skipped = source["customers_skipped"];
	        this.customer_errors = source["customer_errors"];
	        this.opportunities_total = source["opportunities_total"];
	        this.opportunities_imported = source["opportunities_imported"];
	        this.opportunities_skipped = source["opportunities_skipped"];
	        this.opportunity_errors = source["opportunity_errors"];
	        this.payments_total = source["payments_total"];
	        this.payments_imported = source["payments_imported"];
	        this.payments_skipped = source["payments_skipped"];
	        this.payment_errors = source["payment_errors"];
	        this.products_total = source["products_total"];
	        this.products_imported = source["products_imported"];
	        this.products_skipped = source["products_skipped"];
	        this.product_errors = source["product_errors"];
	        this.total_records = source["total_records"];
	        this.total_imported = source["total_imported"];
	        this.duration = source["duration"];
	        this.start_time = this.convertValues(source["start_time"], time.Time);
	        this.end_time = this.convertValues(source["end_time"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace deletion {
	
	export class Request {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    entity_type: string;
	    entity_id: string;
	    entity_label: string;
	    requested_by: string;
	    requested_by_name: string;
	    requested_role: string;
	    reason: string;
	    status: string;
	    reviewed_by: string;
	    reviewed_by_name: string;
	    reviewed_at?: time.Time;
	    review_notes: string;
	
	    static createFrom(source: any = {}) {
	        return new Request(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.entity_type = source["entity_type"];
	        this.entity_id = source["entity_id"];
	        this.entity_label = source["entity_label"];
	        this.requested_by = source["requested_by"];
	        this.requested_by_name = source["requested_by_name"];
	        this.requested_role = source["requested_role"];
	        this.reason = source["reason"];
	        this.status = source["status"];
	        this.reviewed_by = source["reviewed_by"];
	        this.reviewed_by_name = source["reviewed_by_name"];
	        this.reviewed_at = this.convertValues(source["reviewed_at"], time.Time);
	        this.review_notes = source["review_notes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace device {
	
	export class RegistrationResult {
	    device_id: string;
	    machine_id: string;
	    status: string;
	    is_first_setup: boolean;
	    user_id?: string;
	    user_name?: string;
	    role_name?: string;
	    permissions?: string[];
	
	    static createFrom(source: any = {}) {
	        return new RegistrationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.device_id = source["device_id"];
	        this.machine_id = source["machine_id"];
	        this.status = source["status"];
	        this.is_first_setup = source["is_first_setup"];
	        this.user_id = source["user_id"];
	        this.user_name = source["user_name"];
	        this.role_name = source["role_name"];
	        this.permissions = source["permissions"];
	    }
	}

}

export namespace documents {
	
	export class ActionProposalItemVM {
	    action?: string;
	    source_type?: string;
	    label: string;
	    reason: string;
	    priority?: string;
	    required_deterministic_service?: string;
	
	    static createFrom(source: any = {}) {
	        return new ActionProposalItemVM(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.source_type = source["source_type"];
	        this.label = source["label"];
	        this.reason = source["reason"];
	        this.priority = source["priority"];
	        this.required_deterministic_service = source["required_deterministic_service"];
	    }
	}
	export class EvidenceSourceItemVM {
	    source_type?: string;
	    label: string;
	    required?: number;
	    present?: number;
	    missing?: number;
	    confidence?: number;
	    status?: string;
	    priority?: string;
	    last_updated?: string;
	
	    static createFrom(source: any = {}) {
	        return new EvidenceSourceItemVM(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_type = source["source_type"];
	        this.label = source["label"];
	        this.required = source["required"];
	        this.present = source["present"];
	        this.missing = source["missing"];
	        this.confidence = source["confidence"];
	        this.status = source["status"];
	        this.priority = source["priority"];
	        this.last_updated = source["last_updated"];
	    }
	}
	export class IntakeReviewRecordVM {
	    decision: string;
	    reviewStatus: string;
	    actor: string;
	    reason?: string;
	    correlationId: string;
	    createdAt: string;
	    proposedDeterministicService?: string;
	
	    static createFrom(source: any = {}) {
	        return new IntakeReviewRecordVM(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.decision = source["decision"];
	        this.reviewStatus = source["reviewStatus"];
	        this.actor = source["actor"];
	        this.reason = source["reason"];
	        this.correlationId = source["correlationId"];
	        this.createdAt = source["createdAt"];
	        this.proposedDeterministicService = source["proposedDeterministicService"];
	    }
	}
	export class SourceRegistryItemVM {
	    sourceId: string;
	    kind: string;
	    label: string;
	    path?: string;
	    privacyClass: string;
	    processingStatus: string;
	    candidateCount: number;
	    currentCandidate: boolean;
	    auditRefCount: number;
	    lastSeenAtDisplay?: string;
	
	    static createFrom(source: any = {}) {
	        return new SourceRegistryItemVM(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceId = source["sourceId"];
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.path = source["path"];
	        this.privacyClass = source["privacyClass"];
	        this.processingStatus = source["processingStatus"];
	        this.candidateCount = source["candidateCount"];
	        this.currentCandidate = source["currentCandidate"];
	        this.auditRefCount = source["auditRefCount"];
	        this.lastSeenAtDisplay = source["lastSeenAtDisplay"];
	    }
	}
	export class IntakeFieldRowVM {
	    name: string;
	    label: string;
	    value: string;
	    confidenceDisplay?: string;
	    status: shared.StatusBadgeVM;
	    sourceRef?: string;
	    required: boolean;
	
	    static createFrom(source: any = {}) {
	        return new IntakeFieldRowVM(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.label = source["label"];
	        this.value = source["value"];
	        this.confidenceDisplay = source["confidenceDisplay"];
	        this.status = this.convertValues(source["status"], shared.StatusBadgeVM);
	        this.sourceRef = source["sourceRef"];
	        this.required = source["required"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class IntakeCandidateReviewVM {
	    id: string;
	    sourceLabel: string;
	    sourceKind: string;
	    businessObjectType: string;
	    classification: string;
	    reviewStatus: shared.StatusBadgeVM;
	    confidenceDisplay: string;
	    warningCount: number;
	    lastReviewAction?: string;
	    serviceTarget?: string;
	    extractedFields: IntakeFieldRowVM[];
	    sources: EvidenceSourceItemVM[];
	    sourceRegistry?: SourceRegistryItemVM[];
	    actionProposals: ActionProposalItemVM[];
	    auditRefs?: string[];
	    warnings?: string[];
	    lastReview?: IntakeReviewRecordVM;
	    reviewCommands: viewmodel.ActionButton[];
	
	    static createFrom(source: any = {}) {
	        return new IntakeCandidateReviewVM(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sourceLabel = source["sourceLabel"];
	        this.sourceKind = source["sourceKind"];
	        this.businessObjectType = source["businessObjectType"];
	        this.classification = source["classification"];
	        this.reviewStatus = this.convertValues(source["reviewStatus"], shared.StatusBadgeVM);
	        this.confidenceDisplay = source["confidenceDisplay"];
	        this.warningCount = source["warningCount"];
	        this.lastReviewAction = source["lastReviewAction"];
	        this.serviceTarget = source["serviceTarget"];
	        this.extractedFields = this.convertValues(source["extractedFields"], IntakeFieldRowVM);
	        this.sources = this.convertValues(source["sources"], EvidenceSourceItemVM);
	        this.sourceRegistry = this.convertValues(source["sourceRegistry"], SourceRegistryItemVM);
	        this.actionProposals = this.convertValues(source["actionProposals"], ActionProposalItemVM);
	        this.auditRefs = source["auditRefs"];
	        this.warnings = source["warnings"];
	        this.lastReview = this.convertValues(source["lastReview"], IntakeReviewRecordVM);
	        this.reviewCommands = this.convertValues(source["reviewCommands"], viewmodel.ActionButton);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class IntakeCandidateSummaryVM {
	    id: string;
	    sourceLabel: string;
	    sourceKind: string;
	    businessObjectType: string;
	    classification: string;
	    reviewStatus: shared.StatusBadgeVM;
	    confidenceDisplay: string;
	    warningCount: number;
	    lastReviewAction?: string;
	    serviceTarget?: string;
	
	    static createFrom(source: any = {}) {
	        return new IntakeCandidateSummaryVM(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sourceLabel = source["sourceLabel"];
	        this.sourceKind = source["sourceKind"];
	        this.businessObjectType = source["businessObjectType"];
	        this.classification = source["classification"];
	        this.reviewStatus = this.convertValues(source["reviewStatus"], shared.StatusBadgeVM);
	        this.confidenceDisplay = source["confidenceDisplay"];
	        this.warningCount = source["warningCount"];
	        this.lastReviewAction = source["lastReviewAction"];
	        this.serviceTarget = source["serviceTarget"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class KpiStatusItemVM {
	    label: string;
	    value: string;
	    meta?: string;
	    status?: string;
	    priority?: string;
	
	    static createFrom(source: any = {}) {
	        return new KpiStatusItemVM(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.value = source["value"];
	        this.meta = source["meta"];
	        this.status = source["status"];
	        this.priority = source["priority"];
	    }
	}
	export class IntakeReviewVM {
	    queueMetrics: KpiStatusItemVM[];
	    selected?: IntakeCandidateReviewVM;
	    candidates: IntakeCandidateSummaryVM[];
	    actions: viewmodel.ActionButton[];
	
	    static createFrom(source: any = {}) {
	        return new IntakeReviewVM(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.queueMetrics = this.convertValues(source["queueMetrics"], KpiStatusItemVM);
	        this.selected = this.convertValues(source["selected"], IntakeCandidateReviewVM);
	        this.candidates = this.convertValues(source["candidates"], IntakeCandidateSummaryVM);
	        this.actions = this.convertValues(source["actions"], viewmodel.ActionButton);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

export namespace engines {
	
	export class ComplianceData {
	    type: string;
	    data: Record<string, any>;
	    rule_set: string;
	    threshold: number;
	
	    static createFrom(source: any = {}) {
	        return new ComplianceData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.data = source["data"];
	        this.rule_set = source["rule_set"];
	        this.threshold = source["threshold"];
	    }
	}
	export class ComplianceResult {
	    compliant: boolean;
	    score: number;
	    violations: string[];
	    convergence: boolean;
	    iterations: number;
	    recommendations: string[];
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new ComplianceResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.compliant = source["compliant"];
	        this.score = source["score"];
	        this.violations = source["violations"];
	        this.convergence = source["convergence"];
	        this.iterations = source["iterations"];
	        this.recommendations = source["recommendations"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class Customer360 {
	    customer_id: string;
	    business_name: string;
	    total_orders: number;
	    total_value: number;
	    avg_payment_days: number;
	    grade: string;
	    entanglement: Record<string, any>;
	    risk_factors: string[];
	    relation_years: number;
	    last_contact: time.Time;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new Customer360(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.customer_id = source["customer_id"];
	        this.business_name = source["business_name"];
	        this.total_orders = source["total_orders"];
	        this.total_value = source["total_value"];
	        this.avg_payment_days = source["avg_payment_days"];
	        this.grade = source["grade"];
	        this.entanglement = source["entanglement"];
	        this.risk_factors = source["risk_factors"];
	        this.relation_years = source["relation_years"];
	        this.last_contact = this.convertValues(source["last_contact"], time.Time);
	        this.timestamp = source["timestamp"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ERPEvent {
	    type: string;
	    data: Record<string, any>;
	    source: string;
	    priority: number;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new ERPEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.data = source["data"];
	        this.source = source["source"];
	        this.priority = source["priority"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class InvoiceGeometry {
	    id: string;
	    customer_id: string;
	    amount: number;
	    issue_date: time.Time;
	    due_date: time.Time;
	    payment_date: time.Time;
	    status: string;
	    currency: string;
	    item_count: number;
	    discount_applied: number;
	
	    static createFrom(source: any = {}) {
	        return new InvoiceGeometry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.customer_id = source["customer_id"];
	        this.amount = source["amount"];
	        this.issue_date = this.convertValues(source["issue_date"], time.Time);
	        this.due_date = this.convertValues(source["due_date"], time.Time);
	        this.payment_date = this.convertValues(source["payment_date"], time.Time);
	        this.status = source["status"];
	        this.currency = source["currency"];
	        this.item_count = source["item_count"];
	        this.discount_applied = source["discount_applied"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InvoiceResult {
	    invoice_id: string;
	    flow_analysis: Record<string, number>;
	    reconciled: boolean;
	    predicted_days: number;
	    confidence: number;
	    recommendations: string[];
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new InvoiceResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.invoice_id = source["invoice_id"];
	        this.flow_analysis = source["flow_analysis"];
	        this.reconciled = source["reconciled"];
	        this.predicted_days = source["predicted_days"];
	        this.confidence = source["confidence"];
	        this.recommendations = source["recommendations"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class RoutingResult {
	    event_id: string;
	    geometry: string;
	    // Go type: struct { R1 float64 "json:\"r1\""; R2 float64 "json:\"r2\""; R3 float64 "json:\"r3\"" }
	    three_regimes: any;
	    difficulty: number;
	    route_reason: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new RoutingResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.event_id = source["event_id"];
	        this.geometry = source["geometry"];
	        this.three_regimes = this.convertValues(source["three_regimes"], Object);
	        this.difficulty = source["difficulty"];
	        this.route_reason = source["route_reason"];
	        this.timestamp = source["timestamp"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TenderItem {
	    product_id: string;
	    product_name: string;
	    quantity: number;
	    unit_price: number;
	    margin: number;
	
	    static createFrom(source: any = {}) {
	        return new TenderItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.product_id = source["product_id"];
	        this.product_name = source["product_name"];
	        this.quantity = source["quantity"];
	        this.unit_price = source["unit_price"];
	        this.margin = source["margin"];
	    }
	}
	export class TenderGeometry {
	    id: string;
	    customer_id: string;
	    description: string;
	    items: TenderItem[];
	    deadline: time.Time;
	    budget: number;
	    is_abb: boolean;
	    is_emergency: boolean;
	    required_date: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new TenderGeometry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.customer_id = source["customer_id"];
	        this.description = source["description"];
	        this.items = this.convertValues(source["items"], TenderItem);
	        this.deadline = this.convertValues(source["deadline"], time.Time);
	        this.budget = source["budget"];
	        this.is_abb = source["is_abb"];
	        this.is_emergency = source["is_emergency"];
	        this.required_date = this.convertValues(source["required_date"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class TenderResult {
	    tender_id: string;
	    feasible: boolean;
	    optimal_quote: number;
	    margin: number;
	    matched_items: TenderItem[];
	    constraints: number;
	    satisfied: number;
	    recommendation: string;
	    abb_warning: boolean;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new TenderResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tender_id = source["tender_id"];
	        this.feasible = source["feasible"];
	        this.optimal_quote = source["optimal_quote"];
	        this.margin = source["margin"];
	        this.matched_items = this.convertValues(source["matched_items"], TenderItem);
	        this.constraints = source["constraints"];
	        this.satisfied = source["satisfied"];
	        this.recommendation = source["recommendation"];
	        this.abb_warning = source["abb_warning"];
	        this.timestamp = source["timestamp"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WilliamsMetrics {
	    total_items: number;
	    optimal_batch_size: number;
	    memory_mb: number;
	    memory_savings_mb: number;
	    memory_savings_percent: number;
	    efficiency: number;
	    space_complexity: string;
	    proof_url: string;
	
	    static createFrom(source: any = {}) {
	        return new WilliamsMetrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_items = source["total_items"];
	        this.optimal_batch_size = source["optimal_batch_size"];
	        this.memory_mb = source["memory_mb"];
	        this.memory_savings_mb = source["memory_savings_mb"];
	        this.memory_savings_percent = source["memory_savings_percent"];
	        this.efficiency = source["efficiency"];
	        this.space_complexity = source["space_complexity"];
	        this.proof_url = source["proof_url"];
	    }
	}

}

export namespace evidence {
	
	export class ActionProposal {
	    action: string;
	    label: string;
	    reason: string;
	    priority: string;
	    source_type: string;
	    mutates_state: boolean;
	    required_deterministic_service: string;
	
	    static createFrom(source: any = {}) {
	        return new ActionProposal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.label = source["label"];
	        this.reason = source["reason"];
	        this.priority = source["priority"];
	        this.source_type = source["source_type"];
	        this.mutates_state = source["mutates_state"];
	        this.required_deterministic_service = source["required_deterministic_service"];
	    }
	}
	export class AllocationEvidence {
	    allocation_id: string;
	    bank_statement_line_id: string;
	    source_type: string;
	    source_id: string;
	    amount: number;
	    allocation_type: string;
	    confidence: number;
	    allocation_status: string;
	    status: string;
	    priority: string;
	
	    static createFrom(source: any = {}) {
	        return new AllocationEvidence(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.allocation_id = source["allocation_id"];
	        this.bank_statement_line_id = source["bank_statement_line_id"];
	        this.source_type = source["source_type"];
	        this.source_id = source["source_id"];
	        this.amount = source["amount"];
	        this.allocation_type = source["allocation_type"];
	        this.confidence = source["confidence"];
	        this.allocation_status = source["allocation_status"];
	        this.status = source["status"];
	        this.priority = source["priority"];
	    }
	}
	export class AllocationSummary {
	    total_allocations: number;
	    matched: number;
	    partial: number;
	    mixed: number;
	    conflicts: number;
	    unresolved: number;
	    total_amount: number;
	
	    static createFrom(source: any = {}) {
	        return new AllocationSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_allocations = source["total_allocations"];
	        this.matched = source["matched"];
	        this.partial = source["partial"];
	        this.mixed = source["mixed"];
	        this.conflicts = source["conflicts"];
	        this.unresolved = source["unresolved"];
	        this.total_amount = source["total_amount"];
	    }
	}
	export class CashExposure {
	    open_ar: number;
	    overdue_ar: number;
	    due_in_window: number;
	    confirmed_uninvoiced_orders: number;
	    weighted_pipeline: number;
	    total_attention: number;
	    overdue_ratio: number;
	    priority: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new CashExposure(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.open_ar = source["open_ar"];
	        this.overdue_ar = source["overdue_ar"];
	        this.due_in_window = source["due_in_window"];
	        this.confirmed_uninvoiced_orders = source["confirmed_uninvoiced_orders"];
	        this.weighted_pipeline = source["weighted_pipeline"];
	        this.total_attention = source["total_attention"];
	        this.overdue_ratio = source["overdue_ratio"];
	        this.priority = source["priority"];
	        this.status = source["status"];
	    }
	}
	export class PostingReadiness {
	    status: string;
	    priority: string;
	    total_sources: number;
	    missing_journals: number;
	    draft_entries: number;
	    trial_balance_ready: boolean;
	    balanced_account_count: number;
	    imbalanced_account_count: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new PostingReadiness(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.priority = source["priority"];
	        this.total_sources = source["total_sources"];
	        this.missing_journals = source["missing_journals"];
	        this.draft_entries = source["draft_entries"];
	        this.trial_balance_ready = source["trial_balance_ready"];
	        this.balanced_account_count = source["balanced_account_count"];
	        this.imbalanced_account_count = source["imbalanced_account_count"];
	        this.message = source["message"];
	    }
	}
	export class EvidenceSourceStatus {
	    source_type: string;
	    label: string;
	    required: number;
	    present: number;
	    missing: number;
	    confidence: number;
	    status: string;
	    priority: string;
	
	    static createFrom(source: any = {}) {
	        return new EvidenceSourceStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_type = source["source_type"];
	        this.label = source["label"];
	        this.required = source["required"];
	        this.present = source["present"];
	        this.missing = source["missing"];
	        this.confidence = source["confidence"];
	        this.status = source["status"];
	        this.priority = source["priority"];
	    }
	}
	export class TimeWindow {
	    start: time.Time;
	    end: time.Time;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new TimeWindow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start = this.convertValues(source["start"], time.Time);
	        this.end = this.convertValues(source["end"], time.Time);
	        this.label = source["label"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CommandCenter {
	    window: TimeWindow;
	    cash: CashExposure;
	    evidence_sources: EvidenceSourceStatus[];
	    bank_allocations: AllocationEvidence[];
	    allocation_summary: AllocationSummary;
	    posting: PostingReadiness;
	    unmatched_bank_lines: number;
	    unmatched_bank_amount: number;
	    open_follow_up_tasks: number;
	    exportable_audit_items: number;
	    overall_status: string;
	    next_action: string;
	    action_proposals: ActionProposal[];
	
	    static createFrom(source: any = {}) {
	        return new CommandCenter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.window = this.convertValues(source["window"], TimeWindow);
	        this.cash = this.convertValues(source["cash"], CashExposure);
	        this.evidence_sources = this.convertValues(source["evidence_sources"], EvidenceSourceStatus);
	        this.bank_allocations = this.convertValues(source["bank_allocations"], AllocationEvidence);
	        this.allocation_summary = this.convertValues(source["allocation_summary"], AllocationSummary);
	        this.posting = this.convertValues(source["posting"], PostingReadiness);
	        this.unmatched_bank_lines = source["unmatched_bank_lines"];
	        this.unmatched_bank_amount = source["unmatched_bank_amount"];
	        this.open_follow_up_tasks = source["open_follow_up_tasks"];
	        this.exportable_audit_items = source["exportable_audit_items"];
	        this.overall_status = source["overall_status"];
	        this.next_action = source["next_action"];
	        this.action_proposals = this.convertValues(source["action_proposals"], ActionProposal);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	

}

export namespace finance {
	
	export class ARAgingBucket {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    customer_id: string;
	    customer_name: string;
	    snapshot_date: time.Time;
	    less_15_days: number;
	    days_16_30: number;
	    days_31_60: number;
	    days_61_90: number;
	    over_90_days: number;
	    total_outstanding: number;
	    total_overdue: number;
	    risk_tier: string;
	    risk_score: number;
	    overdue_days: number;
	
	    static createFrom(source: any = {}) {
	        return new ARAgingBucket(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.snapshot_date = this.convertValues(source["snapshot_date"], time.Time);
	        this.less_15_days = source["less_15_days"];
	        this.days_16_30 = source["days_16_30"];
	        this.days_31_60 = source["days_31_60"];
	        this.days_61_90 = source["days_61_90"];
	        this.over_90_days = source["over_90_days"];
	        this.total_outstanding = source["total_outstanding"];
	        this.total_overdue = source["total_overdue"];
	        this.risk_tier = source["risk_tier"];
	        this.risk_score = source["risk_score"];
	        this.overdue_days = source["overdue_days"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BalanceGap {
	    from_statement_id: string;
	    to_statement_id: string;
	    from_date: time.Time;
	    to_date: time.Time;
	    closing_balance: number;
	    opening_balance: number;
	    gap_amount: number;
	
	    static createFrom(source: any = {}) {
	        return new BalanceGap(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.from_statement_id = source["from_statement_id"];
	        this.to_statement_id = source["to_statement_id"];
	        this.from_date = this.convertValues(source["from_date"], time.Time);
	        this.to_date = this.convertValues(source["to_date"], time.Time);
	        this.closing_balance = source["closing_balance"];
	        this.opening_balance = source["opening_balance"];
	        this.gap_amount = source["gap_amount"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BalanceContinuityReportData {
	    bank_account_id: string;
	    bank_name: string;
	    gaps: BalanceGap[];
	    total_gap_amount: number;
	    is_continuous: boolean;
	    statements_covered: number;
	
	    static createFrom(source: any = {}) {
	        return new BalanceContinuityReportData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bank_account_id = source["bank_account_id"];
	        this.bank_name = source["bank_name"];
	        this.gaps = this.convertValues(source["gaps"], BalanceGap);
	        this.total_gap_amount = source["total_gap_amount"];
	        this.is_continuous = source["is_continuous"];
	        this.statements_covered = source["statements_covered"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class BankExpenseEntry {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    bank_statement_line_id: string;
	    division: string;
	    expense_date: time.Time;
	    description: string;
	    category: string;
	    amount: number;
	    currency: string;
	    vat_amount: number;
	    gl_account_id?: string;
	    is_posted: boolean;
	    journal_entry_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new BankExpenseEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.bank_statement_line_id = source["bank_statement_line_id"];
	        this.division = source["division"];
	        this.expense_date = this.convertValues(source["expense_date"], time.Time);
	        this.description = source["description"];
	        this.category = source["category"];
	        this.amount = source["amount"];
	        this.currency = source["currency"];
	        this.vat_amount = source["vat_amount"];
	        this.gl_account_id = source["gl_account_id"];
	        this.is_posted = source["is_posted"];
	        this.journal_entry_id = source["journal_entry_id"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BankReconciliationAuditLog {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    bank_statement_id: string;
	    bank_statement_line_id?: string;
	    action: string;
	    action_detail: string;
	    performed_by: string;
	    performed_at: time.Time;
	    is_automatic: boolean;
	    confidence_score: number;
	    reason: string;
	    is_reversed: boolean;
	    reversed_by: string;
	    reversed_at?: time.Time;
	    reversal_reason: string;
	
	    static createFrom(source: any = {}) {
	        return new BankReconciliationAuditLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.bank_statement_id = source["bank_statement_id"];
	        this.bank_statement_line_id = source["bank_statement_line_id"];
	        this.action = source["action"];
	        this.action_detail = source["action_detail"];
	        this.performed_by = source["performed_by"];
	        this.performed_at = this.convertValues(source["performed_at"], time.Time);
	        this.is_automatic = source["is_automatic"];
	        this.confidence_score = source["confidence_score"];
	        this.reason = source["reason"];
	        this.is_reversed = source["is_reversed"];
	        this.reversed_by = source["reversed_by"];
	        this.reversed_at = this.convertValues(source["reversed_at"], time.Time);
	        this.reversal_reason = source["reversal_reason"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BankStatementLine {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    bank_statement_id: string;
	    line_number: number;
	    transaction_date: time.Time;
	    value_date: time.Time;
	    description: string;
	    reference: string;
	    debit: number;
	    credit: number;
	    balance: number;
	    transaction_type: string;
	    category: string;
	    sub_category: string;
	    extracted_customer: string;
	    extracted_supplier: string;
	    extracted_invoices: string;
	    extracted_po_numbers: string;
	    is_matched: boolean;
	    matched_payment_id: string;
	    matched_journal_id: string;
	    matched_invoice_ids: string;
	    matched_expense_id?: string;
	    match_type: string;
	    match_confidence: number;
	    verified_by: string;
	    verified_at?: time.Time;
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new BankStatementLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.bank_statement_id = source["bank_statement_id"];
	        this.line_number = source["line_number"];
	        this.transaction_date = this.convertValues(source["transaction_date"], time.Time);
	        this.value_date = this.convertValues(source["value_date"], time.Time);
	        this.description = source["description"];
	        this.reference = source["reference"];
	        this.debit = source["debit"];
	        this.credit = source["credit"];
	        this.balance = source["balance"];
	        this.transaction_type = source["transaction_type"];
	        this.category = source["category"];
	        this.sub_category = source["sub_category"];
	        this.extracted_customer = source["extracted_customer"];
	        this.extracted_supplier = source["extracted_supplier"];
	        this.extracted_invoices = source["extracted_invoices"];
	        this.extracted_po_numbers = source["extracted_po_numbers"];
	        this.is_matched = source["is_matched"];
	        this.matched_payment_id = source["matched_payment_id"];
	        this.matched_journal_id = source["matched_journal_id"];
	        this.matched_invoice_ids = source["matched_invoice_ids"];
	        this.matched_expense_id = source["matched_expense_id"];
	        this.match_type = source["match_type"];
	        this.match_confidence = source["match_confidence"];
	        this.verified_by = source["verified_by"];
	        this.verified_at = this.convertValues(source["verified_at"], time.Time);
	        this.notes = source["notes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BankStatement {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    bank_account_id: string;
	    statement_number: string;
	    statement_date: time.Time;
	    period_start: time.Time;
	    period_end: time.Time;
	    opening_balance: number;
	    closing_balance: number;
	    currency: string;
	    total_debits: number;
	    total_credits: number;
	    debit_count: number;
	    credit_count: number;
	    status: string;
	    reconciled_at?: time.Time;
	    reconciled_by: string;
	    imported_from: string;
	    import_method: string;
	    ocr_confidence: number;
	    notes: string;
	    division: string;
	    balance_verified: boolean;
	    discrepancy_amount: number;
	    lines?: BankStatementLine[];
	
	    static createFrom(source: any = {}) {
	        return new BankStatement(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.bank_account_id = source["bank_account_id"];
	        this.statement_number = source["statement_number"];
	        this.statement_date = this.convertValues(source["statement_date"], time.Time);
	        this.period_start = this.convertValues(source["period_start"], time.Time);
	        this.period_end = this.convertValues(source["period_end"], time.Time);
	        this.opening_balance = source["opening_balance"];
	        this.closing_balance = source["closing_balance"];
	        this.currency = source["currency"];
	        this.total_debits = source["total_debits"];
	        this.total_credits = source["total_credits"];
	        this.debit_count = source["debit_count"];
	        this.credit_count = source["credit_count"];
	        this.status = source["status"];
	        this.reconciled_at = this.convertValues(source["reconciled_at"], time.Time);
	        this.reconciled_by = source["reconciled_by"];
	        this.imported_from = source["imported_from"];
	        this.import_method = source["import_method"];
	        this.ocr_confidence = source["ocr_confidence"];
	        this.notes = source["notes"];
	        this.division = source["division"];
	        this.balance_verified = source["balance_verified"];
	        this.discrepancy_amount = source["discrepancy_amount"];
	        this.lines = this.convertValues(source["lines"], BankStatementLine);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class BookBankReconciliation {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    bank_account_id: string;
	    reconciliation_date: time.Time;
	    currency: string;
	    bank_statement_balance: number;
	    deposits_in_transit: number;
	    outstanding_cheques: number;
	    bank_errors: number;
	    adjusted_bank_balance: number;
	    book_balance: number;
	    bank_charges_not_recorded: number;
	    interest_not_recorded: number;
	    nsf_cheques: number;
	    book_errors: number;
	    adjusted_book_balance: number;
	    difference: number;
	    is_reconciled: boolean;
	    reconciled_by: string;
	    reconciled_at?: time.Time;
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new BookBankReconciliation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.bank_account_id = source["bank_account_id"];
	        this.reconciliation_date = this.convertValues(source["reconciliation_date"], time.Time);
	        this.currency = source["currency"];
	        this.bank_statement_balance = source["bank_statement_balance"];
	        this.deposits_in_transit = source["deposits_in_transit"];
	        this.outstanding_cheques = source["outstanding_cheques"];
	        this.bank_errors = source["bank_errors"];
	        this.adjusted_bank_balance = source["adjusted_bank_balance"];
	        this.book_balance = source["book_balance"];
	        this.bank_charges_not_recorded = source["bank_charges_not_recorded"];
	        this.interest_not_recorded = source["interest_not_recorded"];
	        this.nsf_cheques = source["nsf_cheques"];
	        this.book_errors = source["book_errors"];
	        this.adjusted_book_balance = source["adjusted_book_balance"];
	        this.difference = source["difference"];
	        this.is_reconciled = source["is_reconciled"];
	        this.reconciled_by = source["reconciled_by"];
	        this.reconciled_at = this.convertValues(source["reconciled_at"], time.Time);
	        this.notes = source["notes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ChartOfAccount {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    account_code: string;
	    account_name: string;
	    account_type: string;
	    balance: number;
	    is_active: boolean;
	    is_vat_account: boolean;
	    vat_direction: string;
	    parent_account_id: string;
	    account_group: string;
	
	    static createFrom(source: any = {}) {
	        return new ChartOfAccount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.account_code = source["account_code"];
	        this.account_name = source["account_name"];
	        this.account_type = source["account_type"];
	        this.balance = source["balance"];
	        this.is_active = source["is_active"];
	        this.is_vat_account = source["is_vat_account"];
	        this.vat_direction = source["vat_direction"];
	        this.parent_account_id = source["parent_account_id"];
	        this.account_group = source["account_group"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ChequeRegister {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    bank_account_id: string;
	    cheque_book_no: string;
	    start_number: number;
	    end_number: number;
	    current_number: number;
	    status: string;
	    issued_date: time.Time;
	    exhausted_date?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new ChequeRegister(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.bank_account_id = source["bank_account_id"];
	        this.cheque_book_no = source["cheque_book_no"];
	        this.start_number = source["start_number"];
	        this.end_number = source["end_number"];
	        this.current_number = source["current_number"];
	        this.status = source["status"];
	        this.issued_date = this.convertValues(source["issued_date"], time.Time);
	        this.exhausted_date = this.convertValues(source["exhausted_date"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CompanyBankAccount {
	    id: string;
	    division: string;
	    bank_name: string;
	    account_name: string;
	    account_number: string;
	    iban: string;
	    swift_bic: string;
	    currency: string;
	    is_active: boolean;
	    display_order: number;
	    booking_rate: number;
	    created_at: time.Time;
	    updated_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new CompanyBankAccount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.division = source["division"];
	        this.bank_name = source["bank_name"];
	        this.account_name = source["account_name"];
	        this.account_number = source["account_number"];
	        this.iban = source["iban"];
	        this.swift_bic = source["swift_bic"];
	        this.currency = source["currency"];
	        this.is_active = source["is_active"];
	        this.display_order = source["display_order"];
	        this.booking_rate = source["booking_rate"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CreditNoteItem {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    credit_note_id: string;
	    line_number: number;
	    description: string;
	    quantity: number;
	    rate: number;
	    total_bhd: number;
	
	    static createFrom(source: any = {}) {
	        return new CreditNoteItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.credit_note_id = source["credit_note_id"];
	        this.line_number = source["line_number"];
	        this.description = source["description"];
	        this.quantity = source["quantity"];
	        this.rate = source["rate"];
	        this.total_bhd = source["total_bhd"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CreditNote {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    credit_note_number: string;
	    credit_note_date: time.Time;
	    invoice_id: string;
	    invoice_number: string;
	    customer_id: string;
	    customer_name: string;
	    reason: string;
	    subtotal_bhd: number;
	    vat_bhd: number;
	    vat_percent: number;
	    grand_total_bhd: number;
	    status: string;
	    division: string;
	    applied_at?: time.Time;
	    credit_note_hash: string;
	    items: CreditNoteItem[];
	
	    static createFrom(source: any = {}) {
	        return new CreditNote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.credit_note_number = source["credit_note_number"];
	        this.credit_note_date = this.convertValues(source["credit_note_date"], time.Time);
	        this.invoice_id = source["invoice_id"];
	        this.invoice_number = source["invoice_number"];
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.reason = source["reason"];
	        this.subtotal_bhd = source["subtotal_bhd"];
	        this.vat_bhd = source["vat_bhd"];
	        this.vat_percent = source["vat_percent"];
	        this.grand_total_bhd = source["grand_total_bhd"];
	        this.status = source["status"];
	        this.division = source["division"];
	        this.applied_at = this.convertValues(source["applied_at"], time.Time);
	        this.credit_note_hash = source["credit_note_hash"];
	        this.items = this.convertValues(source["items"], CreditNoteItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class CurrencyExchangeRate {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    deleted_at: gorm.DeletedAt;
	    currency_code: string;
	    rate: number;
	    effective_from: time.Time;
	    effective_to?: time.Time;
	    set_by: string;
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new CurrencyExchangeRate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.deleted_at = this.convertValues(source["deleted_at"], gorm.DeletedAt);
	        this.currency_code = source["currency_code"];
	        this.rate = source["rate"];
	        this.effective_from = this.convertValues(source["effective_from"], time.Time);
	        this.effective_to = this.convertValues(source["effective_to"], time.Time);
	        this.set_by = source["set_by"];
	        this.notes = source["notes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DBInvoiceItem {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    invoice_id: string;
	    line_number: number;
	    description: string;
	    quantity: number;
	    rate: number;
	    total_bhd: number;
	    product_id: string;
	    product_code: string;
	    equipment: string;
	    model: string;
	    specification: string;
	    detailed_description: string;
	    currency: string;
	    fob: number;
	    freight: number;
	    total_cost: number;
	    margin_percent: number;
	    total_price: number;
	
	    static createFrom(source: any = {}) {
	        return new DBInvoiceItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.invoice_id = source["invoice_id"];
	        this.line_number = source["line_number"];
	        this.description = source["description"];
	        this.quantity = source["quantity"];
	        this.rate = source["rate"];
	        this.total_bhd = source["total_bhd"];
	        this.product_id = source["product_id"];
	        this.product_code = source["product_code"];
	        this.equipment = source["equipment"];
	        this.model = source["model"];
	        this.specification = source["specification"];
	        this.detailed_description = source["detailed_description"];
	        this.currency = source["currency"];
	        this.fob = source["fob"];
	        this.freight = source["freight"];
	        this.total_cost = source["total_cost"];
	        this.margin_percent = source["margin_percent"];
	        this.total_price = source["total_price"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DepositInTransit {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    bank_account_id: string;
	    deposit_date: time.Time;
	    amount: number;
	    currency: string;
	    deposit_slip_no: string;
	    description: string;
	    source_type: string;
	    customer_id?: string;
	    invoice_ids: string;
	    status: string;
	    cleared_date?: time.Time;
	    matched_line_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new DepositInTransit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.bank_account_id = source["bank_account_id"];
	        this.deposit_date = this.convertValues(source["deposit_date"], time.Time);
	        this.amount = source["amount"];
	        this.currency = source["currency"];
	        this.deposit_slip_no = source["deposit_slip_no"];
	        this.description = source["description"];
	        this.source_type = source["source_type"];
	        this.customer_id = source["customer_id"];
	        this.invoice_ids = source["invoice_ids"];
	        this.status = source["status"];
	        this.cleared_date = this.convertValues(source["cleared_date"], time.Time);
	        this.matched_line_id = source["matched_line_id"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class StatementHash {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    bank_account_id: string;
	    statement_hash: string;
	    period_start: time.Time;
	    period_end: time.Time;
	    transaction_count: number;
	    closing_balance: number;
	    imported_at: time.Time;
	    bank_statement_id: string;
	
	    static createFrom(source: any = {}) {
	        return new StatementHash(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.bank_account_id = source["bank_account_id"];
	        this.statement_hash = source["statement_hash"];
	        this.period_start = this.convertValues(source["period_start"], time.Time);
	        this.period_end = this.convertValues(source["period_end"], time.Time);
	        this.transaction_count = source["transaction_count"];
	        this.closing_balance = source["closing_balance"];
	        this.imported_at = this.convertValues(source["imported_at"], time.Time);
	        this.bank_statement_id = source["bank_statement_id"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DuplicateStatementCheck {
	    is_duplicate: boolean;
	    existing?: StatementHash;
	
	    static createFrom(source: any = {}) {
	        return new DuplicateStatementCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.is_duplicate = source["is_duplicate"];
	        this.existing = this.convertValues(source["existing"], StatementHash);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ExpenseCategory {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    name: string;
	    code: string;
	    description: string;
	    gl_account_id?: string;
	    default_tax_rate: number;
	    is_active: boolean;
	    sort_order: number;
	    gl_account_name?: string;
	
	    static createFrom(source: any = {}) {
	        return new ExpenseCategory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.name = source["name"];
	        this.code = source["code"];
	        this.description = source["description"];
	        this.gl_account_id = source["gl_account_id"];
	        this.default_tax_rate = source["default_tax_rate"];
	        this.is_active = source["is_active"];
	        this.sort_order = source["sort_order"];
	        this.gl_account_name = source["gl_account_name"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ExpenseEntry {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    entry_number: string;
	    division: string;
	    expense_date: time.Time;
	    due_date?: time.Time;
	    description: string;
	    category_id: string;
	    vendor_id?: string;
	    source_type: string;
	    source_ref_id?: string;
	    bank_expense_entry_id?: string;
	    project_id?: string;
	    customer_id?: string;
	    opportunity_id?: string;
	    order_id?: string;
	    cost_center: string;
	    currency: string;
	    amount: number;
	    vat_amount: number;
	    total_amount: number;
	    status: string;
	    payment_status: string;
	    submitted_at?: time.Time;
	    submitted_by: string;
	    approved_at?: time.Time;
	    approved_by: string;
	    rejected_at?: time.Time;
	    rejected_by: string;
	    rejection_reason: string;
	    posted_at?: time.Time;
	    posted_by: string;
	    paid_at?: time.Time;
	    payment_method: string;
	    payment_reference: string;
	    bank_account_id?: string;
	    journal_entry_id?: string;
	    notes: string;
	    category_name?: string;
	    vendor_name?: string;
	
	    static createFrom(source: any = {}) {
	        return new ExpenseEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.entry_number = source["entry_number"];
	        this.division = source["division"];
	        this.expense_date = this.convertValues(source["expense_date"], time.Time);
	        this.due_date = this.convertValues(source["due_date"], time.Time);
	        this.description = source["description"];
	        this.category_id = source["category_id"];
	        this.vendor_id = source["vendor_id"];
	        this.source_type = source["source_type"];
	        this.source_ref_id = source["source_ref_id"];
	        this.bank_expense_entry_id = source["bank_expense_entry_id"];
	        this.project_id = source["project_id"];
	        this.customer_id = source["customer_id"];
	        this.opportunity_id = source["opportunity_id"];
	        this.order_id = source["order_id"];
	        this.cost_center = source["cost_center"];
	        this.currency = source["currency"];
	        this.amount = source["amount"];
	        this.vat_amount = source["vat_amount"];
	        this.total_amount = source["total_amount"];
	        this.status = source["status"];
	        this.payment_status = source["payment_status"];
	        this.submitted_at = this.convertValues(source["submitted_at"], time.Time);
	        this.submitted_by = source["submitted_by"];
	        this.approved_at = this.convertValues(source["approved_at"], time.Time);
	        this.approved_by = source["approved_by"];
	        this.rejected_at = this.convertValues(source["rejected_at"], time.Time);
	        this.rejected_by = source["rejected_by"];
	        this.rejection_reason = source["rejection_reason"];
	        this.posted_at = this.convertValues(source["posted_at"], time.Time);
	        this.posted_by = source["posted_by"];
	        this.paid_at = this.convertValues(source["paid_at"], time.Time);
	        this.payment_method = source["payment_method"];
	        this.payment_reference = source["payment_reference"];
	        this.bank_account_id = source["bank_account_id"];
	        this.journal_entry_id = source["journal_entry_id"];
	        this.notes = source["notes"];
	        this.category_name = source["category_name"];
	        this.vendor_name = source["vendor_name"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ExpenseVendor {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    name: string;
	    contact_name: string;
	    email: string;
	    phone: string;
	    payment_terms: string;
	    tax_number: string;
	    notes: string;
	    is_active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ExpenseVendor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.name = source["name"];
	        this.contact_name = source["contact_name"];
	        this.email = source["email"];
	        this.phone = source["phone"];
	        this.payment_terms = source["payment_terms"];
	        this.tax_number = source["tax_number"];
	        this.notes = source["notes"];
	        this.is_active = source["is_active"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FXRate {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    from_currency: string;
	    to_currency: string;
	    rate_date: time.Time;
	    rate: number;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new FXRate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.from_currency = source["from_currency"];
	        this.to_currency = source["to_currency"];
	        this.rate_date = this.convertValues(source["rate_date"], time.Time);
	        this.rate = source["rate"];
	        this.source = source["source"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FXRevaluation {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    bank_account_id: string;
	    revaluation_date: time.Time;
	    foreign_currency: string;
	    foreign_balance: number;
	    previous_rate: number;
	    previous_bhd: number;
	    current_rate: number;
	    current_bhd: number;
	    gain_loss_bhd: number;
	    is_posted: boolean;
	    journal_entry_id?: string;
	    posted_by: string;
	    posted_at?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new FXRevaluation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.bank_account_id = source["bank_account_id"];
	        this.revaluation_date = this.convertValues(source["revaluation_date"], time.Time);
	        this.foreign_currency = source["foreign_currency"];
	        this.foreign_balance = source["foreign_balance"];
	        this.previous_rate = source["previous_rate"];
	        this.previous_bhd = source["previous_bhd"];
	        this.current_rate = source["current_rate"];
	        this.current_bhd = source["current_bhd"];
	        this.gain_loss_bhd = source["gain_loss_bhd"];
	        this.is_posted = source["is_posted"];
	        this.journal_entry_id = source["journal_entry_id"];
	        this.posted_by = source["posted_by"];
	        this.posted_at = this.convertValues(source["posted_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Invoice {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    invoice_number: string;
	    invoice_date: time.Time;
	    customer_id: string;
	    customer_name: string;
	    order_id?: string;
	    customer_po_number: string;
	    grand_total_bhd: number;
	    status: string;
	    outstanding_bhd: number;
	    subtotal_bhd: number;
	    due_date: time.Time;
	    updated_by: string;
	    rfq_id: string;
	    quote_id: string;
	    offer_id: string;
	    offer_number: string;
	    delivery_note_id: string;
	    delivery_note_number: string;
	    total_supplier_cost_bhd: number;
	    gross_margin_bhd: number;
	    gross_margin_percent: number;
	    customer_reference: string;
	    attention_person: string;
	    attention_company: string;
	    attention_phone: string;
	    attention_address: string;
	    delivery_weeks: string;
	    country_of_origin: string;
	    issued_by: string;
	    contact_phone: string;
	    discount_percent: number;
	    payment_terms: string;
	    delivery_terms: string;
	    division: string;
	    field_visibility: string;
	    delivery_note_ref: string;
	    mode_of_payment: string;
	    suppliers_ref: string;
	    other_references: string;
	    buyers_order_number: string;
	    buyers_order_date?: time.Time;
	    despatch_document_no: string;
	    delivery_note_date?: time.Time;
	    despatched_through: string;
	    destination: string;
	    place_of_supply: string;
	    terms_of_delivery: string;
	    vat_bhd: number;
	    vat_percent: number;
	    journal_entry_id: string;
	    invoice_hash: string;
	    notes: string;
	    items: DBInvoiceItem[];
	
	    static createFrom(source: any = {}) {
	        return new Invoice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.invoice_number = source["invoice_number"];
	        this.invoice_date = this.convertValues(source["invoice_date"], time.Time);
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.order_id = source["order_id"];
	        this.customer_po_number = source["customer_po_number"];
	        this.grand_total_bhd = source["grand_total_bhd"];
	        this.status = source["status"];
	        this.outstanding_bhd = source["outstanding_bhd"];
	        this.subtotal_bhd = source["subtotal_bhd"];
	        this.due_date = this.convertValues(source["due_date"], time.Time);
	        this.updated_by = source["updated_by"];
	        this.rfq_id = source["rfq_id"];
	        this.quote_id = source["quote_id"];
	        this.offer_id = source["offer_id"];
	        this.offer_number = source["offer_number"];
	        this.delivery_note_id = source["delivery_note_id"];
	        this.delivery_note_number = source["delivery_note_number"];
	        this.total_supplier_cost_bhd = source["total_supplier_cost_bhd"];
	        this.gross_margin_bhd = source["gross_margin_bhd"];
	        this.gross_margin_percent = source["gross_margin_percent"];
	        this.customer_reference = source["customer_reference"];
	        this.attention_person = source["attention_person"];
	        this.attention_company = source["attention_company"];
	        this.attention_phone = source["attention_phone"];
	        this.attention_address = source["attention_address"];
	        this.delivery_weeks = source["delivery_weeks"];
	        this.country_of_origin = source["country_of_origin"];
	        this.issued_by = source["issued_by"];
	        this.contact_phone = source["contact_phone"];
	        this.discount_percent = source["discount_percent"];
	        this.payment_terms = source["payment_terms"];
	        this.delivery_terms = source["delivery_terms"];
	        this.division = source["division"];
	        this.field_visibility = source["field_visibility"];
	        this.delivery_note_ref = source["delivery_note_ref"];
	        this.mode_of_payment = source["mode_of_payment"];
	        this.suppliers_ref = source["suppliers_ref"];
	        this.other_references = source["other_references"];
	        this.buyers_order_number = source["buyers_order_number"];
	        this.buyers_order_date = this.convertValues(source["buyers_order_date"], time.Time);
	        this.despatch_document_no = source["despatch_document_no"];
	        this.delivery_note_date = this.convertValues(source["delivery_note_date"], time.Time);
	        this.despatched_through = source["despatched_through"];
	        this.destination = source["destination"];
	        this.place_of_supply = source["place_of_supply"];
	        this.terms_of_delivery = source["terms_of_delivery"];
	        this.vat_bhd = source["vat_bhd"];
	        this.vat_percent = source["vat_percent"];
	        this.journal_entry_id = source["journal_entry_id"];
	        this.invoice_hash = source["invoice_hash"];
	        this.notes = source["notes"];
	        this.items = this.convertValues(source["items"], DBInvoiceItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InvoiceFiltersVM {
	    statusOptions: viewmodel.Option[];
	    dateRange?: string;
	    search?: string;
	
	    static createFrom(source: any = {}) {
	        return new InvoiceFiltersVM(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.statusOptions = this.convertValues(source["statusOptions"], viewmodel.Option);
	        this.dateRange = source["dateRange"];
	        this.search = source["search"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InvoiceSummaryVM {
	    totalOutstanding: string;
	    overdueCount: number;
	    overdueAmount: string;
	    paidThisMonth: string;
	    averagePaymentDays: number;
	
	    static createFrom(source: any = {}) {
	        return new InvoiceSummaryVM(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalOutstanding = source["totalOutstanding"];
	        this.overdueCount = source["overdueCount"];
	        this.overdueAmount = source["overdueAmount"];
	        this.paidThisMonth = source["paidThisMonth"];
	        this.averagePaymentDays = source["averagePaymentDays"];
	    }
	}
	export class InvoiceListVM {
	    table: shared.TableVM;
	    summary: InvoiceSummaryVM;
	    filters: InvoiceFiltersVM;
	    actions: viewmodel.ActionButton[];
	
	    static createFrom(source: any = {}) {
	        return new InvoiceListVM(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.table = this.convertValues(source["table"], shared.TableVM);
	        this.summary = this.convertValues(source["summary"], InvoiceSummaryVM);
	        this.filters = this.convertValues(source["filters"], InvoiceFiltersVM);
	        this.actions = this.convertValues(source["actions"], viewmodel.ActionButton);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class JournalLine {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    entry_id: string;
	    account_id: string;
	    account_name: string;
	    debit: number;
	    credit: number;
	    description: string;
	    updated_by: string;
	
	    static createFrom(source: any = {}) {
	        return new JournalLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.entry_id = source["entry_id"];
	        this.account_id = source["account_id"];
	        this.account_name = source["account_name"];
	        this.debit = source["debit"];
	        this.credit = source["credit"];
	        this.description = source["description"];
	        this.updated_by = source["updated_by"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class JournalEntry {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    entry_number: string;
	    entry_date: time.Time;
	    description: string;
	    debit_total: number;
	    credit_total: number;
	    is_posted: boolean;
	    posted_at?: time.Time;
	    posted_by: string;
	    fiscal_year: number;
	    fiscal_period: number;
	    source_type: string;
	    source_id: string;
	    is_auto_generated: boolean;
	    reversed_by_id: string;
	    reverses_id: string;
	    updated_by: string;
	    lines?: JournalLine[];
	
	    static createFrom(source: any = {}) {
	        return new JournalEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.entry_number = source["entry_number"];
	        this.entry_date = this.convertValues(source["entry_date"], time.Time);
	        this.description = source["description"];
	        this.debit_total = source["debit_total"];
	        this.credit_total = source["credit_total"];
	        this.is_posted = source["is_posted"];
	        this.posted_at = this.convertValues(source["posted_at"], time.Time);
	        this.posted_by = source["posted_by"];
	        this.fiscal_year = source["fiscal_year"];
	        this.fiscal_period = source["fiscal_period"];
	        this.source_type = source["source_type"];
	        this.source_id = source["source_id"];
	        this.is_auto_generated = source["is_auto_generated"];
	        this.reversed_by_id = source["reversed_by_id"];
	        this.reverses_id = source["reverses_id"];
	        this.updated_by = source["updated_by"];
	        this.lines = this.convertValues(source["lines"], JournalLine);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class OutstandingCheque {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    bank_account_id: string;
	    cheque_number: string;
	    amount: number;
	    currency: string;
	    issued_date: time.Time;
	    payee_name: string;
	    payee_type: string;
	    supplier_id?: string;
	    purpose: string;
	    status: string;
	    cleared_date?: time.Time;
	    matched_line_id?: string;
	    is_stale: boolean;
	    stale_date?: time.Time;
	    reissued_as?: string;
	
	    static createFrom(source: any = {}) {
	        return new OutstandingCheque(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.bank_account_id = source["bank_account_id"];
	        this.cheque_number = source["cheque_number"];
	        this.amount = source["amount"];
	        this.currency = source["currency"];
	        this.issued_date = this.convertValues(source["issued_date"], time.Time);
	        this.payee_name = source["payee_name"];
	        this.payee_type = source["payee_type"];
	        this.supplier_id = source["supplier_id"];
	        this.purpose = source["purpose"];
	        this.status = source["status"];
	        this.cleared_date = this.convertValues(source["cleared_date"], time.Time);
	        this.matched_line_id = source["matched_line_id"];
	        this.is_stale = source["is_stale"];
	        this.stale_date = this.convertValues(source["stale_date"], time.Time);
	        this.reissued_as = source["reissued_as"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Payment {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    invoice_id: string;
	    invoice_number: string;
	    amount_bhd: number;
	    payment_date: time.Time;
	    payment_method: string;
	    days_to_payment: number;
	    idempotency_key: string;
	    journal_entry_id: string;
	    bank_account_id: string;
	    receipt_id?: string;
	    reference: string;
	    division: string;
	    updated_by: string;
	
	    static createFrom(source: any = {}) {
	        return new Payment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.invoice_id = source["invoice_id"];
	        this.invoice_number = source["invoice_number"];
	        this.amount_bhd = source["amount_bhd"];
	        this.payment_date = this.convertValues(source["payment_date"], time.Time);
	        this.payment_method = source["payment_method"];
	        this.days_to_payment = source["days_to_payment"];
	        this.idempotency_key = source["idempotency_key"];
	        this.journal_entry_id = source["journal_entry_id"];
	        this.bank_account_id = source["bank_account_id"];
	        this.receipt_id = source["receipt_id"];
	        this.reference = source["reference"];
	        this.division = source["division"];
	        this.updated_by = source["updated_by"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RecurringExpense {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    name: string;
	    division: string;
	    description: string;
	    category_id: string;
	    vendor_id?: string;
	    frequency: string;
	    interval_value: number;
	    next_run_date: time.Time;
	    last_generated_at?: time.Time;
	    default_amount: number;
	    default_vat_amount: number;
	    currency: string;
	    cost_center: string;
	    project_id?: string;
	    is_active: boolean;
	    auto_submit: boolean;
	    category_name?: string;
	    vendor_name?: string;
	
	    static createFrom(source: any = {}) {
	        return new RecurringExpense(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.name = source["name"];
	        this.division = source["division"];
	        this.description = source["description"];
	        this.category_id = source["category_id"];
	        this.vendor_id = source["vendor_id"];
	        this.frequency = source["frequency"];
	        this.interval_value = source["interval_value"];
	        this.next_run_date = this.convertValues(source["next_run_date"], time.Time);
	        this.last_generated_at = this.convertValues(source["last_generated_at"], time.Time);
	        this.default_amount = source["default_amount"];
	        this.default_vat_amount = source["default_vat_amount"];
	        this.currency = source["currency"];
	        this.cost_center = source["cost_center"];
	        this.project_id = source["project_id"];
	        this.is_active = source["is_active"];
	        this.auto_submit = source["auto_submit"];
	        this.category_name = source["category_name"];
	        this.vendor_name = source["vendor_name"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class StatementBalanceValidation {
	    is_valid: boolean;
	    discrepancy: number;
	
	    static createFrom(source: any = {}) {
	        return new StatementBalanceValidation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.is_valid = source["is_valid"];
	        this.discrepancy = source["discrepancy"];
	    }
	}
	
	export class SupplierInvoiceItem {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    supplier_invoice_id: string;
	    line_number: number;
	    description: string;
	    quantity: number;
	    unit_price: number;
	    total_price: number;
	    currency: string;
	
	    static createFrom(source: any = {}) {
	        return new SupplierInvoiceItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.supplier_invoice_id = source["supplier_invoice_id"];
	        this.line_number = source["line_number"];
	        this.description = source["description"];
	        this.quantity = source["quantity"];
	        this.unit_price = source["unit_price"];
	        this.total_price = source["total_price"];
	        this.currency = source["currency"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SupplierInvoice {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    supplier_id: string;
	    supplier_name: string;
	    purchase_order_id: string;
	    po_number: string;
	    grn_id: string;
	    order_id: string;
	    invoice_number: string;
	    invoice_date: time.Time;
	    due_date: time.Time;
	    currency: string;
	    exchange_rate: number;
	    subtotal_foreign: number;
	    subtotal_bhd: number;
	    vat_foreign: number;
	    vat_bhd: number;
	    total_foreign: number;
	    total_bhd: number;
	    match_status: string;
	    po_match_ok: boolean;
	    grn_match_ok: boolean;
	    status: string;
	    approved_by: string;
	    approved_at?: time.Time;
	    updated_by: string;
	    payment_status: string;
	    payment_date?: time.Time;
	    payment_ref: string;
	    payment_method: string;
	    outstanding_bhd: number;
	    ocr_document_id: string;
	    ocr_confidence: number;
	    division: string;
	    discrepancy_reason: string;
	    dispute_reason: string;
	    journal_entry_id: string;
	    items: SupplierInvoiceItem[];
	
	    static createFrom(source: any = {}) {
	        return new SupplierInvoice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.supplier_id = source["supplier_id"];
	        this.supplier_name = source["supplier_name"];
	        this.purchase_order_id = source["purchase_order_id"];
	        this.po_number = source["po_number"];
	        this.grn_id = source["grn_id"];
	        this.order_id = source["order_id"];
	        this.invoice_number = source["invoice_number"];
	        this.invoice_date = this.convertValues(source["invoice_date"], time.Time);
	        this.due_date = this.convertValues(source["due_date"], time.Time);
	        this.currency = source["currency"];
	        this.exchange_rate = source["exchange_rate"];
	        this.subtotal_foreign = source["subtotal_foreign"];
	        this.subtotal_bhd = source["subtotal_bhd"];
	        this.vat_foreign = source["vat_foreign"];
	        this.vat_bhd = source["vat_bhd"];
	        this.total_foreign = source["total_foreign"];
	        this.total_bhd = source["total_bhd"];
	        this.match_status = source["match_status"];
	        this.po_match_ok = source["po_match_ok"];
	        this.grn_match_ok = source["grn_match_ok"];
	        this.status = source["status"];
	        this.approved_by = source["approved_by"];
	        this.approved_at = this.convertValues(source["approved_at"], time.Time);
	        this.updated_by = source["updated_by"];
	        this.payment_status = source["payment_status"];
	        this.payment_date = this.convertValues(source["payment_date"], time.Time);
	        this.payment_ref = source["payment_ref"];
	        this.payment_method = source["payment_method"];
	        this.outstanding_bhd = source["outstanding_bhd"];
	        this.ocr_document_id = source["ocr_document_id"];
	        this.ocr_confidence = source["ocr_confidence"];
	        this.division = source["division"];
	        this.discrepancy_reason = source["discrepancy_reason"];
	        this.dispute_reason = source["dispute_reason"];
	        this.journal_entry_id = source["journal_entry_id"];
	        this.items = this.convertValues(source["items"], SupplierInvoiceItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SupplierPayment {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    supplier_invoice_id: string;
	    supplier_id: string;
	    amount_foreign: number;
	    currency: string;
	    exchange_rate: number;
	    amount_bhd: number;
	    payment_date: time.Time;
	    payment_method: string;
	    reference: string;
	    notes: string;
	    payment_number: string;
	    journal_entry_id: string;
	    bank_account_id: string;
	    updated_by: string;
	    division: string;
	    supplier_name: string;
	    invoice_number: string;
	
	    static createFrom(source: any = {}) {
	        return new SupplierPayment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.supplier_invoice_id = source["supplier_invoice_id"];
	        this.supplier_id = source["supplier_id"];
	        this.amount_foreign = source["amount_foreign"];
	        this.currency = source["currency"];
	        this.exchange_rate = source["exchange_rate"];
	        this.amount_bhd = source["amount_bhd"];
	        this.payment_date = this.convertValues(source["payment_date"], time.Time);
	        this.payment_method = source["payment_method"];
	        this.reference = source["reference"];
	        this.notes = source["notes"];
	        this.payment_number = source["payment_number"];
	        this.journal_entry_id = source["journal_entry_id"];
	        this.bank_account_id = source["bank_account_id"];
	        this.updated_by = source["updated_by"];
	        this.division = source["division"];
	        this.supplier_name = source["supplier_name"];
	        this.invoice_number = source["invoice_number"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class VATReturn {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    return_number: string;
	    period_start: time.Time;
	    period_end: time.Time;
	    fiscal_year: number;
	    quarter: number;
	    net_vat: number;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new VATReturn(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.return_number = source["return_number"];
	        this.period_start = this.convertValues(source["period_start"], time.Time);
	        this.period_end = this.convertValues(source["period_end"], time.Time);
	        this.fiscal_year = source["fiscal_year"];
	        this.quarter = source["quarter"];
	        this.net_vat = source["net_vat"];
	        this.status = source["status"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace fx {
	
	export class ExposureReport {
	    currency: string;
	    account_count: number;
	    total_foreign: number;
	    current_rate: number;
	    total_bhd: number;
	    unrealized_gain: number;
	    percent_exposure: number;
	
	    static createFrom(source: any = {}) {
	        return new ExposureReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currency = source["currency"];
	        this.account_count = source["account_count"];
	        this.total_foreign = source["total_foreign"];
	        this.current_rate = source["current_rate"];
	        this.total_bhd = source["total_bhd"];
	        this.unrealized_gain = source["unrealized_gain"];
	        this.percent_exposure = source["percent_exposure"];
	    }
	}
	export class ExposureResult {
	    reports: ExposureReport[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new ExposureResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.reports = this.convertValues(source["reports"], ExposureReport);
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RevaluationBatchResult {
	    revaluations: finance.FXRevaluation[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new RevaluationBatchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.revaluations = this.convertValues(source["revaluations"], finance.FXRevaluation);
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace gorm {
	
	export class DeletedAt {
	    Time: time.Time;
	    Valid: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DeletedAt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Time = this.convertValues(source["Time"], time.Time);
	        this.Valid = source["Valid"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace graph {
	
	export class BuildStats {
	    nodes_created: number;
	    edges_created: number;
	    errors: number;
	    duration: number;
	    nodes_by_type: Record<string, number>;
	    edges_by_type: Record<string, number>;
	    start_time: time.Time;
	    end_time: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new BuildStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodes_created = source["nodes_created"];
	        this.edges_created = source["edges_created"];
	        this.errors = source["errors"];
	        this.duration = source["duration"];
	        this.nodes_by_type = source["nodes_by_type"];
	        this.edges_by_type = source["edges_by_type"];
	        this.start_time = this.convertValues(source["start_time"], time.Time);
	        this.end_time = this.convertValues(source["end_time"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class D3Link {
	    source: string;
	    target: string;
	    type: string;
	    weight: number;
	    properties: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new D3Link(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.target = source["target"];
	        this.type = source["type"];
	        this.weight = source["weight"];
	        this.properties = source["properties"];
	    }
	}
	export class D3Node {
	    id: string;
	    label: string;
	    type: string;
	    size: number;
	    properties: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new D3Node(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.type = source["type"];
	        this.size = source["size"];
	        this.properties = source["properties"];
	    }
	}
	export class GraphStats {
	    total_nodes: number;
	    total_edges: number;
	    nodes_by_type: Record<string, number>;
	    edges_by_type: Record<string, number>;
	    density: number;
	    avg_degree: number;
	    max_degree: number;
	    max_degree_node: string;
	
	    static createFrom(source: any = {}) {
	        return new GraphStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_nodes = source["total_nodes"];
	        this.total_edges = source["total_edges"];
	        this.nodes_by_type = source["nodes_by_type"];
	        this.edges_by_type = source["edges_by_type"];
	        this.density = source["density"];
	        this.avg_degree = source["avg_degree"];
	        this.max_degree = source["max_degree"];
	        this.max_degree_node = source["max_degree_node"];
	    }
	}
	export class GraphData {
	    nodes: D3Node[];
	    links: D3Link[];
	    stats: GraphStats;
	
	    static createFrom(source: any = {}) {
	        return new GraphData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodes = this.convertValues(source["nodes"], D3Node);
	        this.links = this.convertValues(source["links"], D3Link);
	        this.stats = this.convertValues(source["stats"], GraphStats);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GraphEdge {
	    id: number;
	    source_id: number;
	    target_id: number;
	    edge_type: string;
	    properties: number[];
	    weight: number;
	    created_at: time.Time;
	    updated_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new GraphEdge(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source_id = source["source_id"];
	        this.target_id = source["target_id"];
	        this.edge_type = source["edge_type"];
	        this.properties = source["properties"];
	        this.weight = source["weight"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GraphNode {
	    id: number;
	    node_type: string;
	    external_id: string;
	    label: string;
	    properties: number[];
	    created_at: time.Time;
	    updated_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new GraphNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.node_type = source["node_type"];
	        this.external_id = source["external_id"];
	        this.label = source["label"];
	        this.properties = source["properties"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace infra {
	
	export class Alert {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    alert_type: string;
	    severity: string;
	    title: string;
	    message: string;
	    is_active: boolean;
	    is_acknowledged: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Alert(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.alert_type = source["alert_type"];
	        this.severity = source["severity"];
	        this.title = source["title"];
	        this.message = source["message"];
	        this.is_active = source["is_active"];
	        this.is_acknowledged = source["is_acknowledged"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AuditLog {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    user_id: string;
	    action: string;
	    resource: string;
	    resource_id: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.user_id = source["user_id"];
	        this.action = source["action"];
	        this.resource = source["resource"];
	        this.resource_id = source["resource_id"];
	        this.description = source["description"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BackupPolicy {
	    auto_backup_enabled: boolean;
	    frequency_days: number;
	    last_backup_at: string;
	    last_backup_path: string;
	    next_backup_due_at: string;
	    due_now: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BackupPolicy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.auto_backup_enabled = source["auto_backup_enabled"];
	        this.frequency_days = source["frequency_days"];
	        this.last_backup_at = source["last_backup_at"];
	        this.last_backup_path = source["last_backup_path"];
	        this.next_backup_due_at = source["next_backup_due_at"];
	        this.due_now = source["due_now"];
	    }
	}
	export class Device {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    machine_id: string;
	    device_name: string;
	    os_info: string;
	    first_seen_at: time.Time;
	    last_seen_at?: time.Time;
	    status: string;
	    approved_by: string;
	    approved_at?: time.Time;
	    is_admin_device: boolean;
	    notes: string;
	    approver_name?: string;
	
	    static createFrom(source: any = {}) {
	        return new Device(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.machine_id = source["machine_id"];
	        this.device_name = source["device_name"];
	        this.os_info = source["os_info"];
	        this.first_seen_at = this.convertValues(source["first_seen_at"], time.Time);
	        this.last_seen_at = this.convertValues(source["last_seen_at"], time.Time);
	        this.status = source["status"];
	        this.approved_by = source["approved_by"];
	        this.approved_at = this.convertValues(source["approved_at"], time.Time);
	        this.is_admin_device = source["is_admin_device"];
	        this.notes = source["notes"];
	        this.approver_name = source["approver_name"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Role {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    name: string;
	    display_name: string;
	    description: string;
	    permissions: string;
	    is_active: boolean;
	    is_system: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Role(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.name = source["name"];
	        this.display_name = source["display_name"];
	        this.description = source["description"];
	        this.permissions = source["permissions"];
	        this.is_active = source["is_active"];
	        this.is_system = source["is_system"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class User {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    username: string;
	    email: string;
	    role_id: string;
	    full_name: string;
	    display_name: string;
	    department: string;
	    job_title: string;
	    is_active: boolean;
	    last_login_at?: time.Time;
	    password_changed_at?: time.Time;
	    must_change_password: boolean;
	    role_name?: string;
	    role: Role;
	
	    static createFrom(source: any = {}) {
	        return new User(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.username = source["username"];
	        this.email = source["email"];
	        this.role_id = source["role_id"];
	        this.full_name = source["full_name"];
	        this.display_name = source["display_name"];
	        this.department = source["department"];
	        this.job_title = source["job_title"];
	        this.is_active = source["is_active"];
	        this.last_login_at = this.convertValues(source["last_login_at"], time.Time);
	        this.password_changed_at = this.convertValues(source["password_changed_at"], time.Time);
	        this.must_change_password = source["must_change_password"];
	        this.role_name = source["role_name"];
	        this.role = this.convertValues(source["role"], Role);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DeviceUser {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    device_id: string;
	    user_id: string;
	    is_primary: boolean;
	    user: User;
	    device: Device;
	
	    static createFrom(source: any = {}) {
	        return new DeviceUser(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.device_id = source["device_id"];
	        this.user_id = source["user_id"];
	        this.is_primary = source["is_primary"];
	        this.user = this.convertValues(source["user"], User);
	        this.device = this.convertValues(source["device"], Device);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class UserSession {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    user_id: string;
	    token: string;
	    refresh_token: string;
	    access_token_expiry: time.Time;
	    refresh_token_expiry: time.Time;
	    last_activity_at: time.Time;
	    is_active: boolean;
	    invalidated_at?: time.Time;
	    invalidated_reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new UserSession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.user_id = source["user_id"];
	        this.token = source["token"];
	        this.refresh_token = source["refresh_token"];
	        this.access_token_expiry = this.convertValues(source["access_token_expiry"], time.Time);
	        this.refresh_token_expiry = this.convertValues(source["refresh_token_expiry"], time.Time);
	        this.last_activity_at = this.convertValues(source["last_activity_at"], time.Time);
	        this.is_active = source["is_active"];
	        this.invalidated_at = this.convertValues(source["invalidated_at"], time.Time);
	        this.invalidated_reason = source["invalidated_reason"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace intake {
	
	export class AuditRef {
	    type: string;
	    source_id: string;
	    summary: string;
	    timestamp?: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.source_id = source["source_id"];
	        this.summary = source["summary"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class SuggestedLink {
	    id: string;
	    label: string;
	    reason: string;
	    business_object_type: string;
	    required_deterministic_service: string;
	
	    static createFrom(source: any = {}) {
	        return new SuggestedLink(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.reason = source["reason"];
	        this.business_object_type = source["business_object_type"];
	        this.required_deterministic_service = source["required_deterministic_service"];
	    }
	}
	export class ExtractedField {
	    name: string;
	    label: string;
	    value?: string;
	    status: string;
	    confidence?: number;
	    source?: string;
	
	    static createFrom(source: any = {}) {
	        return new ExtractedField(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.label = source["label"];
	        this.value = source["value"];
	        this.status = source["status"];
	        this.confidence = source["confidence"];
	        this.source = source["source"];
	    }
	}
	export class Classification {
	    type: string;
	    method?: string;
	    route_to?: string;
	    reason?: string;
	    keywords?: string[];
	    confidence: number;
	
	    static createFrom(source: any = {}) {
	        return new Classification(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.method = source["method"];
	        this.route_to = source["route_to"];
	        this.reason = source["reason"];
	        this.keywords = source["keywords"];
	        this.confidence = source["confidence"];
	    }
	}
	export class SourceRef {
	    id: string;
	    label: string;
	    path?: string;
	    kind: string;
	    processed_at?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new SourceRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.path = source["path"];
	        this.kind = source["kind"];
	        this.processed_at = this.convertValues(source["processed_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Candidate {
	    id: string;
	    source: SourceRef;
	    source_kind: string;
	    business_object_type: string;
	    classification: Classification;
	    extracted_fields: ExtractedField[];
	    suggested_links: SuggestedLink[];
	    review_status: string;
	    audit_refs: AuditRef[];
	    confidence: number;
	    warnings?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Candidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source = this.convertValues(source["source"], SourceRef);
	        this.source_kind = source["source_kind"];
	        this.business_object_type = source["business_object_type"];
	        this.classification = this.convertValues(source["classification"], Classification);
	        this.extracted_fields = this.convertValues(source["extracted_fields"], ExtractedField);
	        this.suggested_links = this.convertValues(source["suggested_links"], SuggestedLink);
	        this.review_status = source["review_status"];
	        this.audit_refs = this.convertValues(source["audit_refs"], AuditRef);
	        this.confidence = source["confidence"];
	        this.warnings = source["warnings"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ContextPack {
	    candidate_id: string;
	    source_summary: string;
	    source_kind: string;
	    business_object_type: string;
	    classification: Classification;
	    extracted_fields: ExtractedField[];
	    missing_fields: string[];
	    suggested_deterministic_service_targets: string[];
	    review_status: string;
	    warnings?: string[];
	    audit_refs: AuditRef[];
	    allowed_agent_actions: string[];
	    forbidden_agent_actions: string[];
	
	    static createFrom(source: any = {}) {
	        return new ContextPack(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.candidate_id = source["candidate_id"];
	        this.source_summary = source["source_summary"];
	        this.source_kind = source["source_kind"];
	        this.business_object_type = source["business_object_type"];
	        this.classification = this.convertValues(source["classification"], Classification);
	        this.extracted_fields = this.convertValues(source["extracted_fields"], ExtractedField);
	        this.missing_fields = source["missing_fields"];
	        this.suggested_deterministic_service_targets = source["suggested_deterministic_service_targets"];
	        this.review_status = source["review_status"];
	        this.warnings = source["warnings"];
	        this.audit_refs = this.convertValues(source["audit_refs"], AuditRef);
	        this.allowed_agent_actions = source["allowed_agent_actions"];
	        this.forbidden_agent_actions = source["forbidden_agent_actions"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ReviewRecord {
	    id: string;
	    candidate_id: string;
	    source_id: string;
	    decision: string;
	    review_status: string;
	    proposed_deterministic_service?: string;
	    actor: string;
	    reason?: string;
	    correlation_id: string;
	    created_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new ReviewRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.candidate_id = source["candidate_id"];
	        this.source_id = source["source_id"];
	        this.decision = source["decision"];
	        this.review_status = source["review_status"];
	        this.proposed_deterministic_service = source["proposed_deterministic_service"];
	        this.actor = source["actor"];
	        this.reason = source["reason"];
	        this.correlation_id = source["correlation_id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SourceAsset {
	    id: string;
	    kind: string;
	    path?: string;
	    label: string;
	    hash?: string;
	    import_batch_id?: string;
	    privacy_class: string;
	    processing_status: string;
	    candidate_ids?: string[];
	    audit_refs?: AuditRef[];
	    first_seen_at: time.Time;
	    last_seen_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new SourceAsset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.path = source["path"];
	        this.label = source["label"];
	        this.hash = source["hash"];
	        this.import_batch_id = source["import_batch_id"];
	        this.privacy_class = source["privacy_class"];
	        this.processing_status = source["processing_status"];
	        this.candidate_ids = source["candidate_ids"];
	        this.audit_refs = this.convertValues(source["audit_refs"], AuditRef);
	        this.first_seen_at = this.convertValues(source["first_seen_at"], time.Time);
	        this.last_seen_at = this.convertValues(source["last_seen_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ReviewExportBundle {
	    schema_version: string;
	    exported_at: time.Time;
	    candidate: Candidate;
	    context_pack: ContextPack;
	    source_assets?: SourceAsset[];
	    review_records: ReviewRecord[];
	
	    static createFrom(source: any = {}) {
	        return new ReviewExportBundle(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema_version = source["schema_version"];
	        this.exported_at = this.convertValues(source["exported_at"], time.Time);
	        this.candidate = this.convertValues(source["candidate"], Candidate);
	        this.context_pack = this.convertValues(source["context_pack"], ContextPack);
	        this.source_assets = this.convertValues(source["source_assets"], SourceAsset);
	        this.review_records = this.convertValues(source["review_records"], ReviewRecord);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	

}

export namespace integration {
	
	export class ToolStatus {
	    name: string;
	    available: boolean;
	    path: string;
	    version: string;
	    required: boolean;
	    install_url: string;
	    error_message?: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.available = source["available"];
	        this.path = source["path"];
	        this.version = source["version"];
	        this.required = source["required"];
	        this.install_url = source["install_url"];
	        this.error_message = source["error_message"];
	    }
	}
	export class ToolsReport {
	    all_required: boolean;
	    all_optional: boolean;
	    tools: Record<string, ToolStatus>;
	    timestamp: time.Time;
	    summary: string;
	    ready_to_use: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ToolsReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.all_required = source["all_required"];
	        this.all_optional = source["all_optional"];
	        this.tools = this.convertValues(source["tools"], ToolStatus, true);
	        this.timestamp = this.convertValues(source["timestamp"], time.Time);
	        this.summary = source["summary"];
	        this.ready_to_use = source["ready_to_use"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class ARAgingDetail {
	    customer_id: string;
	    customer_name: string;
	    invoice_id: string;
	    invoice_number: string;
	    invoice_date: string;
	    due_date: string;
	    amount: number;
	    days_overdue: number;
	    aging_bucket: string;
	
	    static createFrom(source: any = {}) {
	        return new ARAgingDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.invoice_id = source["invoice_id"];
	        this.invoice_number = source["invoice_number"];
	        this.invoice_date = source["invoice_date"];
	        this.due_date = source["due_date"];
	        this.amount = source["amount"];
	        this.days_overdue = source["days_overdue"];
	        this.aging_bucket = source["aging_bucket"];
	    }
	}
	export class ARAgingReport {
	    current: number;
	    days_30: number;
	    days_60: number;
	    days_90: number;
	    days_120_plus: number;
	    total: number;
	    details?: ARAgingDetail[];
	
	    static createFrom(source: any = {}) {
	        return new ARAgingReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current = source["current"];
	        this.days_30 = source["days_30"];
	        this.days_60 = source["days_60"];
	        this.days_90 = source["days_90"];
	        this.days_120_plus = source["days_120_plus"];
	        this.total = source["total"];
	        this.details = this.convertValues(source["details"], ARAgingDetail);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AccountBalance {
	    account_code: string;
	    account_name: string;
	    balance: number;
	
	    static createFrom(source: any = {}) {
	        return new AccountBalance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.account_code = source["account_code"];
	        this.account_name = source["account_name"];
	        this.balance = source["balance"];
	    }
	}
	export class AgingBucketInvoices {
	    bucket: string;
	    total: number;
	    limit: number;
	    offset: number;
	    total_bhd: number;
	    invoices: finance.Invoice[];
	
	    static createFrom(source: any = {}) {
	        return new AgingBucketInvoices(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bucket = source["bucket"];
	        this.total = source["total"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	        this.total_bhd = source["total_bhd"];
	        this.invoices = this.convertValues(source["invoices"], finance.Invoice);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AlertSummary {
	    active_critical: number;
	    active_warning: number;
	    active_info: number;
	    total_active: number;
	    top_alerts: infra.Alert[];
	
	    static createFrom(source: any = {}) {
	        return new AlertSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.active_critical = source["active_critical"];
	        this.active_warning = source["active_warning"];
	        this.active_info = source["active_info"];
	        this.total_active = source["total_active"];
	        this.top_alerts = this.convertValues(source["top_alerts"], infra.Alert);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AllocationProjectLine {
	    project_id: string;
	    project_name: string;
	    allocation_percent: number;
	
	    static createFrom(source: any = {}) {
	        return new AllocationProjectLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.project_id = source["project_id"];
	        this.project_name = source["project_name"];
	        this.allocation_percent = source["allocation_percent"];
	    }
	}
	export class AllocationSummary {
	    employee_id: string;
	    other_projects_total: number;
	    projects: AllocationProjectLine[];
	
	    static createFrom(source: any = {}) {
	        return new AllocationSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.employee_id = source["employee_id"];
	        this.other_projects_total = source["other_projects_total"];
	        this.projects = this.convertValues(source["projects"], AllocationProjectLine);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AlternativeMargin {
	    margin: number;
	    win_probability: number;
	    risk: string;
	
	    static createFrom(source: any = {}) {
	        return new AlternativeMargin(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.margin = source["margin"];
	        this.win_probability = source["win_probability"];
	        this.risk = source["risk"];
	    }
	}
	export class ApplicationPaths {
	    project_root: string;
	    batch_output: string;
	    test_data: string;
	    report_output: string;
	    asymm_math_root: string;
	
	    static createFrom(source: any = {}) {
	        return new ApplicationPaths(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.project_root = source["project_root"];
	        this.batch_output = source["batch_output"];
	        this.test_data = source["test_data"];
	        this.report_output = source["report_output"];
	        this.asymm_math_root = source["asymm_math_root"];
	    }
	}
	export class ArchaeologyScanSummary {
	    total_files: number;
	    processed_files: number;
	    skipped_files: number;
	    failed_files: number;
	    file_type_breakdown: Record<string, number>;
	    average_confidence: number;
	    total_processing_time: number;
	
	    static createFrom(source: any = {}) {
	        return new ArchaeologyScanSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_files = source["total_files"];
	        this.processed_files = source["processed_files"];
	        this.skipped_files = source["skipped_files"];
	        this.failed_files = source["failed_files"];
	        this.file_type_breakdown = source["file_type_breakdown"];
	        this.average_confidence = source["average_confidence"];
	        this.total_processing_time = source["total_processing_time"];
	    }
	}
	export class ArchivedFileResult {
	    data: number[];
	    file_name: string;
	
	    static createFrom(source: any = {}) {
	        return new ArchivedFileResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = source["data"];
	        this.file_name = source["file_name"];
	    }
	}
	export class AttachmentInfo {
	    file_name: string;
	
	    static createFrom(source: any = {}) {
	        return new AttachmentInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_name = source["file_name"];
	    }
	}
	export class AuthState {
	    is_authenticated: boolean;
	    user_email: string;
	    user_name: string;
	    expires_at: time.Time;
	    scopes: string[];
	
	    static createFrom(source: any = {}) {
	        return new AuthState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.is_authenticated = source["is_authenticated"];
	        this.user_email = source["user_email"];
	        this.user_name = source["user_name"];
	        this.expires_at = this.convertValues(source["expires_at"], time.Time);
	        this.scopes = source["scopes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BalanceSheetReport {
	    as_of_date: string;
	    assets: AccountBalance[];
	    total_assets: number;
	    liabilities: AccountBalance[];
	    total_liabilities: number;
	    equity: AccountBalance[];
	    total_equity: number;
	
	    static createFrom(source: any = {}) {
	        return new BalanceSheetReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.as_of_date = source["as_of_date"];
	        this.assets = this.convertValues(source["assets"], AccountBalance);
	        this.total_assets = source["total_assets"];
	        this.liabilities = this.convertValues(source["liabilities"], AccountBalance);
	        this.total_liabilities = source["total_liabilities"];
	        this.equity = this.convertValues(source["equity"], AccountBalance);
	        this.total_equity = source["total_equity"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BatchOfferResult {
	    total_offers: number;
	    total_files: number;
	    processed_files: number;
	    skipped_files: number;
	    failed_files: number;
	    total_time_seconds: number;
	    average_confidence: number;
	    documents_by_type: Record<string, number>;
	    total_cost_usd: number;
	    gpu_usage_percent: number;
	
	    static createFrom(source: any = {}) {
	        return new BatchOfferResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_offers = source["total_offers"];
	        this.total_files = source["total_files"];
	        this.processed_files = source["processed_files"];
	        this.skipped_files = source["skipped_files"];
	        this.failed_files = source["failed_files"];
	        this.total_time_seconds = source["total_time_seconds"];
	        this.average_confidence = source["average_confidence"];
	        this.documents_by_type = source["documents_by_type"];
	        this.total_cost_usd = source["total_cost_usd"];
	        this.gpu_usage_percent = source["gpu_usage_percent"];
	    }
	}
	export class ParsedRFQEmail {
	    file_path: string;
	    file_name: string;
	    file_size: number;
	    file_mod_time: time.Time;
	    offer_number: string;
	    folder_context: string;
	    subject: string;
	    from: string;
	    from_name: string;
	    to: string[];
	    to_names: string[];
	    cc: string[];
	    date_sent: time.Time;
	    date_received: time.Time;
	    body_text: string;
	    body_html: string;
	    rfq_reference: string;
	    customer_name: string;
	    due_date?: time.Time;
	    project_name: string;
	    extracted_items: string[];
	    attachments: AttachmentInfo[];
	    parsed_at: time.Time;
	    parse_success: boolean;
	    parse_error: string;
	
	    static createFrom(source: any = {}) {
	        return new ParsedRFQEmail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_path = source["file_path"];
	        this.file_name = source["file_name"];
	        this.file_size = source["file_size"];
	        this.file_mod_time = this.convertValues(source["file_mod_time"], time.Time);
	        this.offer_number = source["offer_number"];
	        this.folder_context = source["folder_context"];
	        this.subject = source["subject"];
	        this.from = source["from"];
	        this.from_name = source["from_name"];
	        this.to = source["to"];
	        this.to_names = source["to_names"];
	        this.cc = source["cc"];
	        this.date_sent = this.convertValues(source["date_sent"], time.Time);
	        this.date_received = this.convertValues(source["date_received"], time.Time);
	        this.body_text = source["body_text"];
	        this.body_html = source["body_html"];
	        this.rfq_reference = source["rfq_reference"];
	        this.customer_name = source["customer_name"];
	        this.due_date = this.convertValues(source["due_date"], time.Time);
	        this.project_name = source["project_name"];
	        this.extracted_items = source["extracted_items"];
	        this.attachments = this.convertValues(source["attachments"], AttachmentInfo);
	        this.parsed_at = this.convertValues(source["parsed_at"], time.Time);
	        this.parse_success = source["parse_success"];
	        this.parse_error = source["parse_error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BatchParseResult {
	    base_path: string;
	    total_files: number;
	    parsed_files: number;
	    failed_files: number;
	    emails: ParsedRFQEmail[];
	    duration: number;
	    parsed_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new BatchParseResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.base_path = source["base_path"];
	        this.total_files = source["total_files"];
	        this.parsed_files = source["parsed_files"];
	        this.failed_files = source["failed_files"];
	        this.emails = this.convertValues(source["emails"], ParsedRFQEmail);
	        this.duration = source["duration"];
	        this.parsed_at = this.convertValues(source["parsed_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BatchResult {
	    summary: prediction.BatchSummary;
	    predictions: prediction.PaymentPrediction[];
	
	    static createFrom(source: any = {}) {
	        return new BatchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.summary = this.convertValues(source["summary"], prediction.BatchSummary);
	        this.predictions = this.convertValues(source["predictions"], prediction.PaymentPrediction);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BookBankReconciliationReport {
	    reconciliation: finance.BookBankReconciliation;
	    deposits_in_transit: finance.DepositInTransit[];
	    outstanding_cheques: finance.OutstandingCheque[];
	    bank_account: finance.CompanyBankAccount;
	
	    static createFrom(source: any = {}) {
	        return new BookBankReconciliationReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.reconciliation = this.convertValues(source["reconciliation"], finance.BookBankReconciliation);
	        this.deposits_in_transit = this.convertValues(source["deposits_in_transit"], finance.DepositInTransit);
	        this.outstanding_cheques = this.convertValues(source["outstanding_cheques"], finance.OutstandingCheque);
	        this.bank_account = this.convertValues(source["bank_account"], finance.CompanyBankAccount);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BusinessMemoryContextPackResult {
	    candidate_id: string;
	    context_pack: intake.ContextPack;
	    context_pack_toon: string;
	
	    static createFrom(source: any = {}) {
	        return new BusinessMemoryContextPackResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.candidate_id = source["candidate_id"];
	        this.context_pack = this.convertValues(source["context_pack"], intake.ContextPack);
	        this.context_pack_toon = source["context_pack_toon"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BusinessMemoryReviewDecisionRequest {
	    candidate_id: string;
	    decision: string;
	    actor: string;
	    actor_type?: string;
	    reason?: string;
	    proposed_deterministic_service?: string;
	    correlation_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new BusinessMemoryReviewDecisionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.candidate_id = source["candidate_id"];
	        this.decision = source["decision"];
	        this.actor = source["actor"];
	        this.actor_type = source["actor_type"];
	        this.reason = source["reason"];
	        this.proposed_deterministic_service = source["proposed_deterministic_service"];
	        this.correlation_id = source["correlation_id"];
	    }
	}
	export class BusinessMemoryReviewExportResult {
	    candidate_id: string;
	    bundle: intake.ReviewExportBundle;
	    json: string;
	    toon: string;
	
	    static createFrom(source: any = {}) {
	        return new BusinessMemoryReviewExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.candidate_id = source["candidate_id"];
	        this.bundle = this.convertValues(source["bundle"], intake.ReviewExportBundle);
	        this.json = source["json"];
	        this.toon = source["toon"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BusinessMemoryReviewResult {
	    record: intake.ReviewRecord;
	    queue: documents.IntakeReviewVM;
	    context_pack: intake.ContextPack;
	    context_pack_toon: string;
	
	    static createFrom(source: any = {}) {
	        return new BusinessMemoryReviewResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.record = this.convertValues(source["record"], intake.ReviewRecord);
	        this.queue = this.convertValues(source["queue"], documents.IntakeReviewVM);
	        this.context_pack = this.convertValues(source["context_pack"], intake.ContextPack);
	        this.context_pack_toon = source["context_pack_toon"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ButlerCustomerRequest {
	    business_name: string;
	    customer_type: string;
	    payment_grade: string;
	    city: string;
	    country: string;
	    primary_contact: string;
	    primary_email: string;
	    primary_phone: string;
	    mobile_number: string;
	    industry: string;
	    address_line1: string;
	    trn: string;
	
	    static createFrom(source: any = {}) {
	        return new ButlerCustomerRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.business_name = source["business_name"];
	        this.customer_type = source["customer_type"];
	        this.payment_grade = source["payment_grade"];
	        this.city = source["city"];
	        this.country = source["country"];
	        this.primary_contact = source["primary_contact"];
	        this.primary_email = source["primary_email"];
	        this.primary_phone = source["primary_phone"];
	        this.mobile_number = source["mobile_number"];
	        this.industry = source["industry"];
	        this.address_line1 = source["address_line1"];
	        this.trn = source["trn"];
	    }
	}
	export class ButlerOCRInsight {
	    summary: string;
	    extracted_items: any[];
	    detected_customer: string;
	    detected_project: string;
	    required_deadline: string;
	    confidence: number;
	    suggested_actions: butler.ButlerAction[];
	    document_type: string;
	    metadata: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new ButlerOCRInsight(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.summary = source["summary"];
	        this.extracted_items = source["extracted_items"];
	        this.detected_customer = source["detected_customer"];
	        this.detected_project = source["detected_project"];
	        this.required_deadline = source["required_deadline"];
	        this.confidence = source["confidence"];
	        this.suggested_actions = this.convertValues(source["suggested_actions"], butler.ButlerAction);
	        this.document_type = source["document_type"];
	        this.metadata = source["metadata"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ButlerOfferDraftLineItem {
	    description: string;
	    equipment: string;
	    model: string;
	    specification: string;
	    quantity: number;
	    unit_price_bhd: number;
	    optional: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ButlerOfferDraftLineItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.description = source["description"];
	        this.equipment = source["equipment"];
	        this.model = source["model"];
	        this.specification = source["specification"];
	        this.quantity = source["quantity"];
	        this.unit_price_bhd = source["unit_price_bhd"];
	        this.optional = source["optional"];
	    }
	}
	export class ButlerOfferDraftRequest {
	    division: string;
	    prepared_by: string;
	    customer_id: string;
	    customer_name: string;
	    contact_person: string;
	    rfq_reference: string;
	    delivery_terms: string;
	    payment_terms: string;
	    est_delivery: string;
	    country_of_origin: string;
	    quote_type: string;
	    vat_rate: number;
	    line_items: ButlerOfferDraftLineItem[];
	
	    static createFrom(source: any = {}) {
	        return new ButlerOfferDraftRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.division = source["division"];
	        this.prepared_by = source["prepared_by"];
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.contact_person = source["contact_person"];
	        this.rfq_reference = source["rfq_reference"];
	        this.delivery_terms = source["delivery_terms"];
	        this.payment_terms = source["payment_terms"];
	        this.est_delivery = source["est_delivery"];
	        this.country_of_origin = source["country_of_origin"];
	        this.quote_type = source["quote_type"];
	        this.vat_rate = source["vat_rate"];
	        this.line_items = this.convertValues(source["line_items"], ButlerOfferDraftLineItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ButlerSupplierRequest {
	    supplier_name: string;
	    supplier_type: string;
	    country: string;
	    primary_contact: string;
	    email: string;
	    phone: string;
	    address: string;
	    tax_id: string;
	    brands_handled: string;
	    lead_time_days: number;
	
	    static createFrom(source: any = {}) {
	        return new ButlerSupplierRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supplier_name = source["supplier_name"];
	        this.supplier_type = source["supplier_type"];
	        this.country = source["country"];
	        this.primary_contact = source["primary_contact"];
	        this.email = source["email"];
	        this.phone = source["phone"];
	        this.address = source["address"];
	        this.tax_id = source["tax_id"];
	        this.brands_handled = source["brands_handled"];
	        this.lead_time_days = source["lead_time_days"];
	    }
	}
	export class CustomerMetricCard {
	    id: string;
	    business_name: string;
	    customer_type: string;
	    payment_grade: string;
	    total_revenue: number;
	    active_invoices: number;
	    outstanding_bhd: number;
	    overdue_bhd: number;
	    last_order_date?: time.Time;
	    city: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomerMetricCard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.business_name = source["business_name"];
	        this.customer_type = source["customer_type"];
	        this.payment_grade = source["payment_grade"];
	        this.total_revenue = source["total_revenue"];
	        this.active_invoices = source["active_invoices"];
	        this.outstanding_bhd = source["outstanding_bhd"];
	        this.overdue_bhd = source["overdue_bhd"];
	        this.last_order_date = this.convertValues(source["last_order_date"], time.Time);
	        this.city = source["city"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CRMCustomerDashboard {
	    total_customers: number;
	    active_customers: number;
	    total_revenue: number;
	    revenue_yoy: number;
	    total_outstanding: number;
	    overdue_amount: number;
	    overdue_pct: number;
	    top_customers: CustomerMetricCard[];
	    grade_a_count: number;
	    grade_a_revenue: number;
	    grade_b_count: number;
	    grade_b_revenue: number;
	    grade_c_count: number;
	    grade_c_revenue: number;
	    grade_d_count: number;
	    grade_d_revenue: number;
	    top3_revenue_pct: number;
	    top5_revenue_pct: number;
	    top10_revenue_pct: number;
	    customers: CustomerMetricCard[];
	
	    static createFrom(source: any = {}) {
	        return new CRMCustomerDashboard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_customers = source["total_customers"];
	        this.active_customers = source["active_customers"];
	        this.total_revenue = source["total_revenue"];
	        this.revenue_yoy = source["revenue_yoy"];
	        this.total_outstanding = source["total_outstanding"];
	        this.overdue_amount = source["overdue_amount"];
	        this.overdue_pct = source["overdue_pct"];
	        this.top_customers = this.convertValues(source["top_customers"], CustomerMetricCard);
	        this.grade_a_count = source["grade_a_count"];
	        this.grade_a_revenue = source["grade_a_revenue"];
	        this.grade_b_count = source["grade_b_count"];
	        this.grade_b_revenue = source["grade_b_revenue"];
	        this.grade_c_count = source["grade_c_count"];
	        this.grade_c_revenue = source["grade_c_revenue"];
	        this.grade_d_count = source["grade_d_count"];
	        this.grade_d_revenue = source["grade_d_revenue"];
	        this.top3_revenue_pct = source["top3_revenue_pct"];
	        this.top5_revenue_pct = source["top5_revenue_pct"];
	        this.top10_revenue_pct = source["top10_revenue_pct"];
	        this.customers = this.convertValues(source["customers"], CustomerMetricCard);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SupplierMetricCard {
	    id: string;
	    supplier_name: string;
	    supplier_type: string;
	    rating: number;
	    total_purchases: number;
	    active_pos: number;
	    outstanding_bhd: number;
	    overdue_bhd: number;
	    brands_handled: string;
	    country: string;
	
	    static createFrom(source: any = {}) {
	        return new SupplierMetricCard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.supplier_name = source["supplier_name"];
	        this.supplier_type = source["supplier_type"];
	        this.rating = source["rating"];
	        this.total_purchases = source["total_purchases"];
	        this.active_pos = source["active_pos"];
	        this.outstanding_bhd = source["outstanding_bhd"];
	        this.overdue_bhd = source["overdue_bhd"];
	        this.brands_handled = source["brands_handled"];
	        this.country = source["country"];
	    }
	}
	export class CRMSupplierDashboard {
	    total_suppliers: number;
	    active_suppliers: number;
	    total_purchases: number;
	    outstanding_payables: number;
	    overdue_payables: number;
	    top_suppliers: SupplierMetricCard[];
	    suppliers: SupplierMetricCard[];
	
	    static createFrom(source: any = {}) {
	        return new CRMSupplierDashboard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_suppliers = source["total_suppliers"];
	        this.active_suppliers = source["active_suppliers"];
	        this.total_purchases = source["total_purchases"];
	        this.outstanding_payables = source["outstanding_payables"];
	        this.overdue_payables = source["overdue_payables"];
	        this.top_suppliers = this.convertValues(source["top_suppliers"], SupplierMetricCard);
	        this.suppliers = this.convertValues(source["suppliers"], SupplierMetricCard);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CashFlowProjectionDay {
	    date: time.Time;
	    expected_inflows: number;
	    expected_outflows: number;
	    net_cash_flow: number;
	    cumulative_cash: number;
	    opening_balance: number;
	    closing_balance: number;
	
	    static createFrom(source: any = {}) {
	        return new CashFlowProjectionDay(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = this.convertValues(source["date"], time.Time);
	        this.expected_inflows = source["expected_inflows"];
	        this.expected_outflows = source["expected_outflows"];
	        this.net_cash_flow = source["net_cash_flow"];
	        this.cumulative_cash = source["cumulative_cash"];
	        this.opening_balance = source["opening_balance"];
	        this.closing_balance = source["closing_balance"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CashFlowProjection {
	    start_date: time.Time;
	    end_date: time.Time;
	    opening_cash: number;
	    total_inflows: number;
	    total_outflows: number;
	    projected_cash: number;
	    daily_projections: CashFlowProjectionDay[];
	
	    static createFrom(source: any = {}) {
	        return new CashFlowProjection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start_date = this.convertValues(source["start_date"], time.Time);
	        this.end_date = this.convertValues(source["end_date"], time.Time);
	        this.opening_cash = source["opening_cash"];
	        this.total_inflows = source["total_inflows"];
	        this.total_outflows = source["total_outflows"];
	        this.projected_cash = source["projected_cash"];
	        this.daily_projections = this.convertValues(source["daily_projections"], CashFlowProjectionDay);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class CashflowEvidenceProposalReview {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    proposal_key: string;
	    action: string;
	    label: string;
	    reason: string;
	    priority: string;
	    source_type: string;
	    mutates_state: boolean;
	    required_deterministic_service: string;
	    status: string;
	    review_note: string;
	    reviewed_by: string;
	    reviewed_at?: time.Time;
	    window_label: string;
	    window_start: time.Time;
	    window_end: time.Time;
	    last_seen_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new CashflowEvidenceProposalReview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.proposal_key = source["proposal_key"];
	        this.action = source["action"];
	        this.label = source["label"];
	        this.reason = source["reason"];
	        this.priority = source["priority"];
	        this.source_type = source["source_type"];
	        this.mutates_state = source["mutates_state"];
	        this.required_deterministic_service = source["required_deterministic_service"];
	        this.status = source["status"];
	        this.review_note = source["review_note"];
	        this.reviewed_by = source["reviewed_by"];
	        this.reviewed_at = this.convertValues(source["reviewed_at"], time.Time);
	        this.window_label = source["window_label"];
	        this.window_start = this.convertValues(source["window_start"], time.Time);
	        this.window_end = this.convertValues(source["window_end"], time.Time);
	        this.last_seen_at = this.convertValues(source["last_seen_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ChatResponse {
	    response: string;
	    conversation_id: string;
	    actions: butler.ButlerAction[];
	    confidence: number;
	    tokens_used: number;
	    metadata: butler.ButlerResponseMetadata;
	
	    static createFrom(source: any = {}) {
	        return new ChatResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.response = source["response"];
	        this.conversation_id = source["conversation_id"];
	        this.actions = this.convertValues(source["actions"], butler.ButlerAction);
	        this.confidence = source["confidence"];
	        this.tokens_used = source["tokens_used"];
	        this.metadata = this.convertValues(source["metadata"], butler.ButlerResponseMetadata);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ClassificationResult {
	    document_type: string;
	    confidence: number;
	    method: string;
	    route_to: string;
	    suggested_action: string;
	    keywords_found: string[];
	    explanation: string;
	
	    static createFrom(source: any = {}) {
	        return new ClassificationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.document_type = source["document_type"];
	        this.confidence = source["confidence"];
	        this.method = source["method"];
	        this.route_to = source["route_to"];
	        this.suggested_action = source["suggested_action"];
	        this.keywords_found = source["keywords_found"];
	        this.explanation = source["explanation"];
	    }
	}
	export class CollaborativePendingOperation {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    entity_type: string;
	    entity_id: string;
	    operation: string;
	    payload: string;
	    status: string;
	    attempts: number;
	    last_attempt_at?: time.Time;
	    next_attempt_at?: time.Time;
	    error_message: string;
	
	    static createFrom(source: any = {}) {
	        return new CollaborativePendingOperation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.entity_type = source["entity_type"];
	        this.entity_id = source["entity_id"];
	        this.operation = source["operation"];
	        this.payload = source["payload"];
	        this.status = source["status"];
	        this.attempts = source["attempts"];
	        this.last_attempt_at = this.convertValues(source["last_attempt_at"], time.Time);
	        this.next_attempt_at = this.convertValues(source["next_attempt_at"], time.Time);
	        this.error_message = source["error_message"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CompleteFlowResult {
	    tender_id: string;
	    customer_id: string;
	    customer_grade: string;
	    customer_risk: boolean;
	    tender_feasible: boolean;
	    quote_amount: number;
	    margin: number;
	    recommendation: string;
	    predicted_payment_days: number;
	    flow_confidence: number;
	    final_decision: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new CompleteFlowResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tender_id = source["tender_id"];
	        this.customer_id = source["customer_id"];
	        this.customer_grade = source["customer_grade"];
	        this.customer_risk = source["customer_risk"];
	        this.tender_feasible = source["tender_feasible"];
	        this.quote_amount = source["quote_amount"];
	        this.margin = source["margin"];
	        this.recommendation = source["recommendation"];
	        this.predicted_payment_days = source["predicted_payment_days"];
	        this.flow_confidence = source["flow_confidence"];
	        this.final_decision = source["final_decision"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class CostingSheetAttachmentSummary {
	    id: string;
	    scope_id: string;
	    costing_number: string;
	    file_name: string;
	    file_ext: string;
	    mime_type: string;
	    file_size: number;
	    file_hash: string;
	    storage_mode: string;
	    local_path: string;
	    notes: string;
	    uploaded_by: string;
	    created_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new CostingSheetAttachmentSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.scope_id = source["scope_id"];
	        this.costing_number = source["costing_number"];
	        this.file_name = source["file_name"];
	        this.file_ext = source["file_ext"];
	        this.mime_type = source["mime_type"];
	        this.file_size = source["file_size"];
	        this.file_hash = source["file_hash"];
	        this.storage_mode = source["storage_mode"];
	        this.local_path = source["local_path"];
	        this.notes = source["notes"];
	        this.uploaded_by = source["uploaded_by"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CostingExportLineItem {
	    slNo: number;
	    supplier: string;
	    equipment: string;
	    model: string;
	    serialNumber: string;
	    longCode: string;
	    specification: string;
	    detailedDescription: string;
	    currency: string;
	    quantity: number;
	    fob: number;
	    freight: number;
	    freightPercent: number;
	    totalCost: number;
	    marginPercent: number;
	    markupPercent: number;
	    suggestedPrice: number;
	    totalPrice: number;
	    exchangeRate: number;
	    fobBHD: number;
	    freightBHD: number;
	    insurance: number;
	    customsPercent: number;
	    customsBHD: number;
	    handlingPercent: number;
	    handlingBHD: number;
	    financePercent: number;
	    financeBHD: number;
	    otherCosts: number;
	    userPrice: number;
	    userPriceSet: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CostingExportLineItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slNo = source["slNo"];
	        this.supplier = source["supplier"];
	        this.equipment = source["equipment"];
	        this.model = source["model"];
	        this.serialNumber = source["serialNumber"];
	        this.longCode = source["longCode"];
	        this.specification = source["specification"];
	        this.detailedDescription = source["detailedDescription"];
	        this.currency = source["currency"];
	        this.quantity = source["quantity"];
	        this.fob = source["fob"];
	        this.freight = source["freight"];
	        this.freightPercent = source["freightPercent"];
	        this.totalCost = source["totalCost"];
	        this.marginPercent = source["marginPercent"];
	        this.markupPercent = source["markupPercent"];
	        this.suggestedPrice = source["suggestedPrice"];
	        this.totalPrice = source["totalPrice"];
	        this.exchangeRate = source["exchangeRate"];
	        this.fobBHD = source["fobBHD"];
	        this.freightBHD = source["freightBHD"];
	        this.insurance = source["insurance"];
	        this.customsPercent = source["customsPercent"];
	        this.customsBHD = source["customsBHD"];
	        this.handlingPercent = source["handlingPercent"];
	        this.handlingBHD = source["handlingBHD"];
	        this.financePercent = source["financePercent"];
	        this.financeBHD = source["financeBHD"];
	        this.otherCosts = source["otherCosts"];
	        this.userPrice = source["userPrice"];
	        this.userPriceSet = source["userPriceSet"];
	    }
	}
	export class CostingExportData {
	    division: string;
	    source: string;
	    offerId: string;
	    offerNumber: string;
	    date: string;
	    preparedBy: string;
	    customerId: string;
	    customerName: string;
	    contactPerson: string;
	    rfqReference: string;
	    folderNumber: string;
	    costingId: string;
	    subject: string;
	    estDelivery: string;
	    deliveryTerms: string;
	    paymentTerms: string;
	    orderType: string;
	    countryOfOrigin: string;
	    cocCoo: string;
	    testCertificate: string;
	    installation: string;
	    commissioning: string;
	    testing: string;
	    quoteType: string;
	    vatRate: number;
	    hiddenCharges: number;
	    placeOfSupply: string;
	    taxCategory: string;
	    customerTRN: string;
	    body: string;
	    lineItems: CostingExportLineItem[];
	    subtotal: number;
	    discount: number;
	    netAmount: number;
	    vat: number;
	    grandTotal: number;
	    totalCost: number;
	    profit: number;
	    profitPercent: number;
	    opportunityId: number;
	    opportunityRecordId: string;
	    projectName: string;
	    termsAndConditions: string;
	    attachmentScopeId: string;
	    attachments: CostingSheetAttachmentSummary[];
	
	    static createFrom(source: any = {}) {
	        return new CostingExportData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.division = source["division"];
	        this.source = source["source"];
	        this.offerId = source["offerId"];
	        this.offerNumber = source["offerNumber"];
	        this.date = source["date"];
	        this.preparedBy = source["preparedBy"];
	        this.customerId = source["customerId"];
	        this.customerName = source["customerName"];
	        this.contactPerson = source["contactPerson"];
	        this.rfqReference = source["rfqReference"];
	        this.folderNumber = source["folderNumber"];
	        this.costingId = source["costingId"];
	        this.subject = source["subject"];
	        this.estDelivery = source["estDelivery"];
	        this.deliveryTerms = source["deliveryTerms"];
	        this.paymentTerms = source["paymentTerms"];
	        this.orderType = source["orderType"];
	        this.countryOfOrigin = source["countryOfOrigin"];
	        this.cocCoo = source["cocCoo"];
	        this.testCertificate = source["testCertificate"];
	        this.installation = source["installation"];
	        this.commissioning = source["commissioning"];
	        this.testing = source["testing"];
	        this.quoteType = source["quoteType"];
	        this.vatRate = source["vatRate"];
	        this.hiddenCharges = source["hiddenCharges"];
	        this.placeOfSupply = source["placeOfSupply"];
	        this.taxCategory = source["taxCategory"];
	        this.customerTRN = source["customerTRN"];
	        this.body = source["body"];
	        this.lineItems = this.convertValues(source["lineItems"], CostingExportLineItem);
	        this.subtotal = source["subtotal"];
	        this.discount = source["discount"];
	        this.netAmount = source["netAmount"];
	        this.vat = source["vat"];
	        this.grandTotal = source["grandTotal"];
	        this.totalCost = source["totalCost"];
	        this.profit = source["profit"];
	        this.profitPercent = source["profitPercent"];
	        this.opportunityId = source["opportunityId"];
	        this.opportunityRecordId = source["opportunityRecordId"];
	        this.projectName = source["projectName"];
	        this.termsAndConditions = source["termsAndConditions"];
	        this.attachmentScopeId = source["attachmentScopeId"];
	        this.attachments = this.convertValues(source["attachments"], CostingSheetAttachmentSummary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class CostingLineItem {
	    description: string;
	    product_code: string;
	    product_type: string;
	    quantity: number;
	    unit_cost_bhd: number;
	    margin_percent: number;
	
	    static createFrom(source: any = {}) {
	        return new CostingLineItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.description = source["description"];
	        this.product_code = source["product_code"];
	        this.product_type = source["product_type"];
	        this.quantity = source["quantity"];
	        this.unit_cost_bhd = source["unit_cost_bhd"];
	        this.margin_percent = source["margin_percent"];
	    }
	}
	export class CostingLineResult {
	    description: string;
	    product_code: string;
	    product_type: string;
	    quantity: number;
	    unit_cost_bhd: number;
	    margin_percent: number;
	    total_cost_bhd: number;
	    unit_sell_bhd: number;
	    total_sell_bhd: number;
	    unit_profit_bhd: number;
	    total_profit_bhd: number;
	    actual_margin_pct: number;
	
	    static createFrom(source: any = {}) {
	        return new CostingLineResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.description = source["description"];
	        this.product_code = source["product_code"];
	        this.product_type = source["product_type"];
	        this.quantity = source["quantity"];
	        this.unit_cost_bhd = source["unit_cost_bhd"];
	        this.margin_percent = source["margin_percent"];
	        this.total_cost_bhd = source["total_cost_bhd"];
	        this.unit_sell_bhd = source["unit_sell_bhd"];
	        this.total_sell_bhd = source["total_sell_bhd"];
	        this.unit_profit_bhd = source["unit_profit_bhd"];
	        this.total_profit_bhd = source["total_profit_bhd"];
	        this.actual_margin_pct = source["actual_margin_pct"];
	    }
	}
	export class CostingRequest {
	    customer_id: number;
	    opportunity_id?: number;
	    items: CostingLineItem[];
	    apply_discount: boolean;
	    requested_discount?: number;
	    notes?: string;
	
	    static createFrom(source: any = {}) {
	        return new CostingRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.customer_id = source["customer_id"];
	        this.opportunity_id = source["opportunity_id"];
	        this.items = this.convertValues(source["items"], CostingLineItem);
	        this.apply_discount = source["apply_discount"];
	        this.requested_discount = source["requested_discount"];
	        this.notes = source["notes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CostingResult {
	    customer_id: number;
	    customer_name: string;
	    customer_grade: string;
	    opportunity_id?: number;
	    items: CostingLineResult[];
	    total_cost_bhd: number;
	    total_sell_bhd: number;
	    total_discount_bhd: number;
	    total_final_bhd: number;
	    total_profit_bhd: number;
	    standard_margin_pct: number;
	    actual_margin_pct: number;
	    payment_terms: string;
	    advance_required: number;
	    approval_status: string;
	    risk_warnings: string[];
	    recommended_action: string;
	    needs_approval: boolean;
	    valid_until: string;
	    calculated_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new CostingResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.customer_grade = source["customer_grade"];
	        this.opportunity_id = source["opportunity_id"];
	        this.items = this.convertValues(source["items"], CostingLineResult);
	        this.total_cost_bhd = source["total_cost_bhd"];
	        this.total_sell_bhd = source["total_sell_bhd"];
	        this.total_discount_bhd = source["total_discount_bhd"];
	        this.total_final_bhd = source["total_final_bhd"];
	        this.total_profit_bhd = source["total_profit_bhd"];
	        this.standard_margin_pct = source["standard_margin_pct"];
	        this.actual_margin_pct = source["actual_margin_pct"];
	        this.payment_terms = source["payment_terms"];
	        this.advance_required = source["advance_required"];
	        this.approval_status = source["approval_status"];
	        this.risk_warnings = source["risk_warnings"];
	        this.recommended_action = source["recommended_action"];
	        this.needs_approval = source["needs_approval"];
	        this.valid_until = source["valid_until"];
	        this.calculated_at = this.convertValues(source["calculated_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class CostingSheetData {
	    id: number;
	    rfq_id: number;
	    rfq_name: string;
	    opportunity_id: string;
	    revision_number: number;
	    parent_costing_id?: number;
	    is_active: boolean;
	    items: string;
	    subtotal: number;
	    total_markup: number;
	    final_price: number;
	    margin_percent: number;
	    status: string;
	    created_by: string;
	    approved_by: string;
	    approval_required: boolean;
	    risk_warnings: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    offer_number: string;
	    customer_name: string;
	    product_type: string;
	    total_value_bhd: number;
	    line_item_count: number;
	    source_file_path: string;
	    extracted_at?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new CostingSheetData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.rfq_id = source["rfq_id"];
	        this.rfq_name = source["rfq_name"];
	        this.opportunity_id = source["opportunity_id"];
	        this.revision_number = source["revision_number"];
	        this.parent_costing_id = source["parent_costing_id"];
	        this.is_active = source["is_active"];
	        this.items = source["items"];
	        this.subtotal = source["subtotal"];
	        this.total_markup = source["total_markup"];
	        this.final_price = source["final_price"];
	        this.margin_percent = source["margin_percent"];
	        this.status = source["status"];
	        this.created_by = source["created_by"];
	        this.approved_by = source["approved_by"];
	        this.approval_required = source["approval_required"];
	        this.risk_warnings = source["risk_warnings"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.offer_number = source["offer_number"];
	        this.customer_name = source["customer_name"];
	        this.product_type = source["product_type"];
	        this.total_value_bhd = source["total_value_bhd"];
	        this.line_item_count = source["line_item_count"];
	        this.source_file_path = source["source_file_path"];
	        this.extracted_at = this.convertValues(source["extracted_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CreditNoteItemInput {
	    description: string;
	    quantity: number;
	    rate: number;
	
	    static createFrom(source: any = {}) {
	        return new CreditNoteItemInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.description = source["description"];
	        this.quantity = source["quantity"];
	        this.rate = source["rate"];
	    }
	}
	export class CurrentEmployeeContext {
	    employee_id: string;
	    employee_name: string;
	    license_key: string;
	    license_role: string;
	    device_id: string;
	    user_id: string;
	    resolved_by: string;
	    permissions: string[];
	
	    static createFrom(source: any = {}) {
	        return new CurrentEmployeeContext(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.employee_id = source["employee_id"];
	        this.employee_name = source["employee_name"];
	        this.license_key = source["license_key"];
	        this.license_role = source["license_role"];
	        this.device_id = source["device_id"];
	        this.user_id = source["user_id"];
	        this.resolved_by = source["resolved_by"];
	        this.permissions = source["permissions"];
	    }
	}
	export class OrderSummary {
	    order_number: string;
	    order_date: time.Time;
	    total_value_bhd: number;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new OrderSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.order_number = source["order_number"];
	        this.order_date = this.convertValues(source["order_date"], time.Time);
	        this.total_value_bhd = source["total_value_bhd"];
	        this.status = source["status"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OpportunitySummary {
	    id: number;
	    project: string;
	    value: number;
	    status: string;
	    created_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new OpportunitySummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.project = source["project"];
	        this.value = source["value"];
	        this.status = source["status"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PaymentHistoryEntry {
	    payment_date: time.Time;
	    amount_bhd: number;
	    invoice_number: string;
	    days_to_payment: number;
	    payment_method: string;
	
	    static createFrom(source: any = {}) {
	        return new PaymentHistoryEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.payment_date = this.convertValues(source["payment_date"], time.Time);
	        this.amount_bhd = source["amount_bhd"];
	        this.invoice_number = source["invoice_number"];
	        this.days_to_payment = source["days_to_payment"];
	        this.payment_method = source["payment_method"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ReceivablesAgingSummary {
	    current: number;
	    days_30_60: number;
	    days_60_90: number;
	    days_90_120: number;
	    days_120_plus: number;
	    total_outstanding: number;
	
	    static createFrom(source: any = {}) {
	        return new ReceivablesAgingSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current = source["current"];
	        this.days_30_60 = source["days_30_60"];
	        this.days_60_90 = source["days_60_90"];
	        this.days_90_120 = source["days_90_120"];
	        this.days_120_plus = source["days_120_plus"];
	        this.total_outstanding = source["total_outstanding"];
	    }
	}
	export class Customer360Data {
	    customer_id: string;
	    business_name: string;
	    customer_type: string;
	    industry: string;
	    city: string;
	    country: string;
	    relation_years: number;
	    current_grade: string;
	    payment_terms_days: number;
	    avg_payment_days: number;
	    dispute_count: number;
	    is_credit_blocked: boolean;
	    requires_prepayment: boolean;
	    r1: number;
	    r2: number;
	    r3: number;
	    total_orders_value: number;
	    total_orders_count: number;
	    avg_order_value: number;
	    last_order_date?: time.Time;
	    has_abb_competition: boolean;
	    is_emergency_only: boolean;
	    recent_predictions: butler.PredictionRecord[];
	    receivables_aging: ReceivablesAgingSummary;
	    payment_history: PaymentHistoryEntry[];
	    open_opportunities: OpportunitySummary[];
	    recent_orders: OrderSummary[];
	    customer_lifetime_value: number;
	
	    static createFrom(source: any = {}) {
	        return new Customer360Data(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.customer_id = source["customer_id"];
	        this.business_name = source["business_name"];
	        this.customer_type = source["customer_type"];
	        this.industry = source["industry"];
	        this.city = source["city"];
	        this.country = source["country"];
	        this.relation_years = source["relation_years"];
	        this.current_grade = source["current_grade"];
	        this.payment_terms_days = source["payment_terms_days"];
	        this.avg_payment_days = source["avg_payment_days"];
	        this.dispute_count = source["dispute_count"];
	        this.is_credit_blocked = source["is_credit_blocked"];
	        this.requires_prepayment = source["requires_prepayment"];
	        this.r1 = source["r1"];
	        this.r2 = source["r2"];
	        this.r3 = source["r3"];
	        this.total_orders_value = source["total_orders_value"];
	        this.total_orders_count = source["total_orders_count"];
	        this.avg_order_value = source["avg_order_value"];
	        this.last_order_date = this.convertValues(source["last_order_date"], time.Time);
	        this.has_abb_competition = source["has_abb_competition"];
	        this.is_emergency_only = source["is_emergency_only"];
	        this.recent_predictions = this.convertValues(source["recent_predictions"], butler.PredictionRecord);
	        this.receivables_aging = this.convertValues(source["receivables_aging"], ReceivablesAgingSummary);
	        this.payment_history = this.convertValues(source["payment_history"], PaymentHistoryEntry);
	        this.open_opportunities = this.convertValues(source["open_opportunities"], OpportunitySummary);
	        this.recent_orders = this.convertValues(source["recent_orders"], OrderSummary);
	        this.customer_lifetime_value = source["customer_lifetime_value"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GraphMetrics {
	    total_nodes: number;
	    total_edges: number;
	    average_connections: number;
	    graph_density: number;
	
	    static createFrom(source: any = {}) {
	        return new GraphMetrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_nodes = source["total_nodes"];
	        this.total_edges = source["total_edges"];
	        this.average_connections = source["average_connections"];
	        this.graph_density = source["graph_density"];
	    }
	}
	export class GraphRelation {
	    id: string;
	    source: string;
	    target: string;
	    type: string;
	    label: string;
	    weight?: number;
	
	    static createFrom(source: any = {}) {
	        return new GraphRelation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source = source["source"];
	        this.target = source["target"];
	        this.type = source["type"];
	        this.label = source["label"];
	        this.weight = source["weight"];
	    }
	}
	export class GraphPosition {
	    x: number;
	    y: number;
	
	    static createFrom(source: any = {}) {
	        return new GraphPosition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	    }
	}
	export class GraphEntity {
	    id: string;
	    type: string;
	    label: string;
	    data: Record<string, any>;
	    position?: GraphPosition;
	
	    static createFrom(source: any = {}) {
	        return new GraphEntity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.label = source["label"];
	        this.data = source["data"];
	        this.position = this.convertValues(source["position"], GraphPosition);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Customer360Graph {
	    customer_id: string;
	    entities: GraphEntity[];
	    relations: GraphRelation[];
	    metrics: GraphMetrics;
	
	    static createFrom(source: any = {}) {
	        return new Customer360Graph(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.customer_id = source["customer_id"];
	        this.entities = this.convertValues(source["entities"], GraphEntity);
	        this.relations = this.convertValues(source["relations"], GraphRelation);
	        this.metrics = this.convertValues(source["metrics"], GraphMetrics);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InvoiceSummary {
	    id: string;
	    invoice_number: string;
	    invoice_date: time.Time;
	    grand_total_bhd: number;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new InvoiceSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.invoice_number = source["invoice_number"];
	        this.invoice_date = this.convertValues(source["invoice_date"], time.Time);
	        this.grand_total_bhd = source["grand_total_bhd"];
	        this.status = source["status"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CustomerFullProfile {
	    id: string;
	    customer_id: string;
	    business_name: string;
	    customer_type: string;
	    short_code: string;
	    trn: string;
	    industry: string;
	    address_line1: string;
	    city: string;
	    country: string;
	    contacts: crm.CustomerContact[];
	    payment_grade: string;
	    payment_terms_days: number;
	    credit_limit: number;
	    is_credit_blocked: boolean;
	    total_revenue: number;
	    total_orders: number;
	    avg_order_value: number;
	    last_order_date?: time.Time;
	    relation_years: number;
	    outstanding_bhd: number;
	    overdue_bhd: number;
	    ar_aging_buckets: ReceivablesAgingSummary;
	    rfqs_floated: number;
	    rfqs_won: number;
	    win_rate: number;
	    recent_rfqs: OpportunitySummary[];
	    recent_orders: OrderSummary[];
	    recent_invoices: InvoiceSummary[];
	    payment_history: PaymentHistoryEntry[];
	    notes: crm.EntityNote[];
	
	    static createFrom(source: any = {}) {
	        return new CustomerFullProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.customer_id = source["customer_id"];
	        this.business_name = source["business_name"];
	        this.customer_type = source["customer_type"];
	        this.short_code = source["short_code"];
	        this.trn = source["trn"];
	        this.industry = source["industry"];
	        this.address_line1 = source["address_line1"];
	        this.city = source["city"];
	        this.country = source["country"];
	        this.contacts = this.convertValues(source["contacts"], crm.CustomerContact);
	        this.payment_grade = source["payment_grade"];
	        this.payment_terms_days = source["payment_terms_days"];
	        this.credit_limit = source["credit_limit"];
	        this.is_credit_blocked = source["is_credit_blocked"];
	        this.total_revenue = source["total_revenue"];
	        this.total_orders = source["total_orders"];
	        this.avg_order_value = source["avg_order_value"];
	        this.last_order_date = this.convertValues(source["last_order_date"], time.Time);
	        this.relation_years = source["relation_years"];
	        this.outstanding_bhd = source["outstanding_bhd"];
	        this.overdue_bhd = source["overdue_bhd"];
	        this.ar_aging_buckets = this.convertValues(source["ar_aging_buckets"], ReceivablesAgingSummary);
	        this.rfqs_floated = source["rfqs_floated"];
	        this.rfqs_won = source["rfqs_won"];
	        this.win_rate = source["win_rate"];
	        this.recent_rfqs = this.convertValues(source["recent_rfqs"], OpportunitySummary);
	        this.recent_orders = this.convertValues(source["recent_orders"], OrderSummary);
	        this.recent_invoices = this.convertValues(source["recent_invoices"], InvoiceSummary);
	        this.payment_history = this.convertValues(source["payment_history"], PaymentHistoryEntry);
	        this.notes = this.convertValues(source["notes"], crm.EntityNote);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CustomerMatchResult {
	    customer_id: string;
	    business_name: string;
	    short_code: string;
	    score: number;
	    match_reason: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomerMatchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.customer_id = source["customer_id"];
	        this.business_name = source["business_name"];
	        this.short_code = source["short_code"];
	        this.score = source["score"];
	        this.match_reason = source["match_reason"];
	    }
	}
	
	export class ProductSummary {
	    product_id: string;
	    product_code: string;
	    product_name: string;
	    order_count: number;
	    total_quantity: number;
	    total_value_bhd: number;
	
	    static createFrom(source: any = {}) {
	        return new ProductSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.product_id = source["product_id"];
	        this.product_code = source["product_code"];
	        this.product_name = source["product_name"];
	        this.order_count = source["order_count"];
	        this.total_quantity = source["total_quantity"];
	        this.total_value_bhd = source["total_value_bhd"];
	    }
	}
	export class CustomerOrderHistorySummary {
	    avg_order_value: number;
	    order_frequency: number;
	    last_order_date?: time.Time;
	    preferred_products: ProductSummary[];
	
	    static createFrom(source: any = {}) {
	        return new CustomerOrderHistorySummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.avg_order_value = source["avg_order_value"];
	        this.order_frequency = source["order_frequency"];
	        this.last_order_date = this.convertValues(source["last_order_date"], time.Time);
	        this.preferred_products = this.convertValues(source["preferred_products"], ProductSummary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CustomerReceipt {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    receipt_number: string;
	    customer_id: string;
	    customer_name: string;
	    division: string;
	    receipt_date: time.Time;
	    amount_bhd: number;
	    applied_amount_bhd: number;
	    unapplied_amount_bhd: number;
	    payment_method: string;
	    reference: string;
	    status: string;
	    notes: string;
	    updated_by: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomerReceipt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.receipt_number = source["receipt_number"];
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.division = source["division"];
	        this.receipt_date = this.convertValues(source["receipt_date"], time.Time);
	        this.amount_bhd = source["amount_bhd"];
	        this.applied_amount_bhd = source["applied_amount_bhd"];
	        this.unapplied_amount_bhd = source["unapplied_amount_bhd"];
	        this.payment_method = source["payment_method"];
	        this.reference = source["reference"];
	        this.status = source["status"];
	        this.notes = source["notes"];
	        this.updated_by = source["updated_by"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CustomerReceiptAllocation {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    receipt_id: string;
	    invoice_id: string;
	    invoice_number: string;
	    payment_id: string;
	    allocated_amount_bhd: number;
	    allocation_date: time.Time;
	    status: string;
	    updated_by: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomerReceiptAllocation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.receipt_id = source["receipt_id"];
	        this.invoice_id = source["invoice_id"];
	        this.invoice_number = source["invoice_number"];
	        this.payment_id = source["payment_id"];
	        this.allocated_amount_bhd = source["allocated_amount_bhd"];
	        this.allocation_date = this.convertValues(source["allocation_date"], time.Time);
	        this.status = source["status"];
	        this.updated_by = source["updated_by"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CustomerReceiptInput {
	    customer_id: string;
	    customer_name: string;
	    invoice_id: string;
	    amount_bhd: number;
	    receipt_date: string;
	    payment_method: string;
	    reference: string;
	    division: string;
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomerReceiptInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.invoice_id = source["invoice_id"];
	        this.amount_bhd = source["amount_bhd"];
	        this.receipt_date = source["receipt_date"];
	        this.payment_method = source["payment_method"];
	        this.reference = source["reference"];
	        this.division = source["division"];
	        this.notes = source["notes"];
	    }
	}
	export class CustomerRelatedProduct {
	    product_code: string;
	    product_name: string;
	    product_category: string;
	    supplier_name: string;
	    total_quantity: number;
	    order_count: number;
	    last_ordered?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new CustomerRelatedProduct(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.product_code = source["product_code"];
	        this.product_name = source["product_name"];
	        this.product_category = source["product_category"];
	        this.supplier_name = source["supplier_name"];
	        this.total_quantity = source["total_quantity"];
	        this.order_count = source["order_count"];
	        this.last_ordered = this.convertValues(source["last_ordered"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CustomerRelatedSupplier {
	    supplier_id: string;
	    supplier_name: string;
	    supplier_code: string;
	    supplier_type: string;
	    product_count: number;
	    last_ordered?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new CustomerRelatedSupplier(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supplier_id = source["supplier_id"];
	        this.supplier_name = source["supplier_name"];
	        this.supplier_code = source["supplier_code"];
	        this.supplier_type = source["supplier_type"];
	        this.product_count = source["product_count"];
	        this.last_ordered = this.convertValues(source["last_ordered"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CustomerRevenueData {
	    customer_id: string;
	    customer_name: string;
	    revenue: number;
	    invoice_count: number;
	
	    static createFrom(source: any = {}) {
	        return new CustomerRevenueData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.revenue = source["revenue"];
	        this.invoice_count = source["invoice_count"];
	    }
	}
	export class CustomerTypeItem {
	    type: string;
	    label: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new CustomerTypeItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.label = source["label"];
	        this.count = source["count"];
	    }
	}
	export class DBSyncResult {
	    success: boolean;
	    records_pushed: number;
	    records_pulled: number;
	    tables_processed: number;
	    duration: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new DBSyncResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.records_pushed = source["records_pushed"];
	        this.records_pulled = source["records_pulled"];
	        this.tables_processed = source["tables_processed"];
	        this.duration = source["duration"];
	        this.error = source["error"];
	    }
	}
	export class DBSyncSettings {
	    auto_sync_enabled: boolean;
	    sync_frequency_min: number;
	    last_sync_at?: string;
	    last_sync_status?: string;
	    records_synced?: number;
	
	    static createFrom(source: any = {}) {
	        return new DBSyncSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.auto_sync_enabled = source["auto_sync_enabled"];
	        this.sync_frequency_min = source["sync_frequency_min"];
	        this.last_sync_at = source["last_sync_at"];
	        this.last_sync_status = source["last_sync_status"];
	        this.records_synced = source["records_synced"];
	    }
	}
	export class DNItemInputWithSerials {
	    order_item_id: string;
	    ship_qty: number;
	    serial_nos: string[];
	
	    static createFrom(source: any = {}) {
	        return new DNItemInputWithSerials(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.order_item_id = source["order_item_id"];
	        this.ship_qty = source["ship_qty"];
	        this.serial_nos = source["serial_nos"];
	    }
	}
	export class DashboardEvent {
	    id: number;
	    type: string;
	    title: string;
	    time: time.Time;
	    time_ago: string;
	    color: string;
	
	    static createFrom(source: any = {}) {
	        return new DashboardEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.title = source["title"];
	        this.time = this.convertValues(source["time"], time.Time);
	        this.time_ago = source["time_ago"];
	        this.color = source["color"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DashboardStats {
	    active_rfqs: number;
	    active_orders: number;
	    pending_review: number;
	    urgent_count: number;
	    avg_velocity_days: number;
	    win_rate: number;
	    total_revenue: number;
	    month_growth: number;
	    system_health: string;
	    runway_months: number;
	    outstanding_ar: number;
	    ar_days_overdue: number;
	    pending_invoices: number;
	    active_customers: number;
	    revenue_meta: string;
	    activity_year: number;
	    pipeline_value_bhd: number;
	    collection_rate: number;
	    cash_balance_bhd: number;
	    cash_position_note: string;
	    fresh_start_date: string;
	
	    static createFrom(source: any = {}) {
	        return new DashboardStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.active_rfqs = source["active_rfqs"];
	        this.active_orders = source["active_orders"];
	        this.pending_review = source["pending_review"];
	        this.urgent_count = source["urgent_count"];
	        this.avg_velocity_days = source["avg_velocity_days"];
	        this.win_rate = source["win_rate"];
	        this.total_revenue = source["total_revenue"];
	        this.month_growth = source["month_growth"];
	        this.system_health = source["system_health"];
	        this.runway_months = source["runway_months"];
	        this.outstanding_ar = source["outstanding_ar"];
	        this.ar_days_overdue = source["ar_days_overdue"];
	        this.pending_invoices = source["pending_invoices"];
	        this.active_customers = source["active_customers"];
	        this.revenue_meta = source["revenue_meta"];
	        this.activity_year = source["activity_year"];
	        this.pipeline_value_bhd = source["pipeline_value_bhd"];
	        this.collection_rate = source["collection_rate"];
	        this.cash_balance_bhd = source["cash_balance_bhd"];
	        this.cash_position_note = source["cash_position_note"];
	        this.fresh_start_date = source["fresh_start_date"];
	    }
	}
	export class DashboardStatsV2Field {
	    key: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new DashboardStatsV2Field(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	    }
	}
	export class DashboardStatsV2 {
	    stats: DashboardStats;
	    proto_schema: string;
	    fields: DashboardStatsV2Field[];
	
	    static createFrom(source: any = {}) {
	        return new DashboardStatsV2(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stats = this.convertValues(source["stats"], DashboardStats);
	        this.proto_schema = source["proto_schema"];
	        this.fields = this.convertValues(source["fields"], DashboardStatsV2Field);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class DataDiscrepancy {
	    type: string;
	    offer_number: string;
	    invoice_number: string;
	    field: string;
	    extracted_value: string;
	    database_value: string;
	    severity: string;
	    confidence: number;
	    timestamp: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new DataDiscrepancy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.offer_number = source["offer_number"];
	        this.invoice_number = source["invoice_number"];
	        this.field = source["field"];
	        this.extracted_value = source["extracted_value"];
	        this.database_value = source["database_value"];
	        this.severity = source["severity"];
	        this.confidence = source["confidence"];
	        this.timestamp = this.convertValues(source["timestamp"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DataQualityIssue {
	    id: string;
	    severity: string;
	    issue_type: string;
	    entity_type: string;
	    entity_id: string;
	    summary: string;
	    detail: string;
	    primary_action: string;
	    review_status: string;
	    review_note: string;
	    reviewed_by: string;
	    reviewed_at: string;
	
	    static createFrom(source: any = {}) {
	        return new DataQualityIssue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.severity = source["severity"];
	        this.issue_type = source["issue_type"];
	        this.entity_type = source["entity_type"];
	        this.entity_id = source["entity_id"];
	        this.summary = source["summary"];
	        this.detail = source["detail"];
	        this.primary_action = source["primary_action"];
	        this.review_status = source["review_status"];
	        this.review_note = source["review_note"];
	        this.reviewed_by = source["reviewed_by"];
	        this.reviewed_at = source["reviewed_at"];
	    }
	}
	export class DataQualityReview {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    issue_id: string;
	    issue_type: string;
	    severity: string;
	    entity_type: string;
	    entity_id: string;
	    summary: string;
	    detail: string;
	    primary_action: string;
	    status: string;
	    review_note: string;
	    reviewed_by_id: string;
	    reviewed_by: string;
	    reviewed_at?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new DataQualityReview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.issue_id = source["issue_id"];
	        this.issue_type = source["issue_type"];
	        this.severity = source["severity"];
	        this.entity_type = source["entity_type"];
	        this.entity_id = source["entity_id"];
	        this.summary = source["summary"];
	        this.detail = source["detail"];
	        this.primary_action = source["primary_action"];
	        this.status = source["status"];
	        this.review_note = source["review_note"];
	        this.reviewed_by_id = source["reviewed_by_id"];
	        this.reviewed_by = source["reviewed_by"];
	        this.reviewed_at = this.convertValues(source["reviewed_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DealDocumentStatus {
	    document: string;
	    present: boolean;
	    serial?: string;
	    date?: time.Time;
	    record_id?: string;
	    record_type?: string;
	
	    static createFrom(source: any = {}) {
	        return new DealDocumentStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.document = source["document"];
	        this.present = source["present"];
	        this.serial = source["serial"];
	        this.date = this.convertValues(source["date"], time.Time);
	        this.record_id = source["record_id"];
	        this.record_type = source["record_type"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DealTimelineNode {
	    stage: string;
	    serial: string;
	    date: time.Time;
	    state: string;
	    record_id?: string;
	    record_type?: string;
	    count?: number;
	
	    static createFrom(source: any = {}) {
	        return new DealTimelineNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stage = source["stage"];
	        this.serial = source["serial"];
	        this.date = this.convertValues(source["date"], time.Time);
	        this.state = source["state"];
	        this.record_id = source["record_id"];
	        this.record_type = source["record_type"];
	        this.count = source["count"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DealTimeline {
	    order_id: string;
	    nodes: DealTimelineNode[];
	    documents: DealDocumentStatus[];
	
	    static createFrom(source: any = {}) {
	        return new DealTimeline(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.order_id = source["order_id"];
	        this.nodes = this.convertValues(source["nodes"], DealTimelineNode);
	        this.documents = this.convertValues(source["documents"], DealDocumentStatus);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class DeleteCascadeResult {
	    rfq_id: number;
	    linked_costing_sheets: number;
	    linked_offers: number;
	    deleted_costing_sheets: number;
	    deleted_offers: number;
	
	    static createFrom(source: any = {}) {
	        return new DeleteCascadeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rfq_id = source["rfq_id"];
	        this.linked_costing_sheets = source["linked_costing_sheets"];
	        this.linked_offers = source["linked_offers"];
	        this.deleted_costing_sheets = source["deleted_costing_sheets"];
	        this.deleted_offers = source["deleted_offers"];
	    }
	}
	export class DeliveryNoteHeaderInput {
	    delivery_date: time.Time;
	    delivery_address: string;
	    contact_person: string;
	    contact_phone: string;
	    driver_name: string;
	    vehicle_number: string;
	    transport_method: string;
	
	    static createFrom(source: any = {}) {
	        return new DeliveryNoteHeaderInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.delivery_date = this.convertValues(source["delivery_date"], time.Time);
	        this.delivery_address = source["delivery_address"];
	        this.contact_person = source["contact_person"];
	        this.contact_phone = source["contact_phone"];
	        this.driver_name = source["driver_name"];
	        this.vehicle_number = source["vehicle_number"];
	        this.transport_method = source["transport_method"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DeliveryNoteItemInput {
	    order_item_id: string;
	    ship_qty: number;
	
	    static createFrom(source: any = {}) {
	        return new DeliveryNoteItemInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.order_item_id = source["order_item_id"];
	        this.ship_qty = source["ship_qty"];
	    }
	}
	export class DeliveryPlanningItem {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    order_id: string;
	    customer_id: string;
	    dn_number: string;
	    delivery_date: time.Time;
	    delivery_address: string;
	    contact_person: string;
	    contact_phone: string;
	    driver_name: string;
	    vehicle_number: string;
	    transport_method: string;
	    status: string;
	    updated_by: string;
	    signed_by: string;
	    signed_at?: time.Time;
	    signature_image: string;
	    is_partial_delivery: boolean;
	    delivery_sequence: number;
	    total_deliveries: number;
	    items?: crm.DeliveryNoteItem[];
	    customer_area: string;
	    urgency_score: number;
	    days_until_due: number;
	    order_value: number;
	    is_high_priority: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DeliveryPlanningItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.order_id = source["order_id"];
	        this.customer_id = source["customer_id"];
	        this.dn_number = source["dn_number"];
	        this.delivery_date = this.convertValues(source["delivery_date"], time.Time);
	        this.delivery_address = source["delivery_address"];
	        this.contact_person = source["contact_person"];
	        this.contact_phone = source["contact_phone"];
	        this.driver_name = source["driver_name"];
	        this.vehicle_number = source["vehicle_number"];
	        this.transport_method = source["transport_method"];
	        this.status = source["status"];
	        this.updated_by = source["updated_by"];
	        this.signed_by = source["signed_by"];
	        this.signed_at = this.convertValues(source["signed_at"], time.Time);
	        this.signature_image = source["signature_image"];
	        this.is_partial_delivery = source["is_partial_delivery"];
	        this.delivery_sequence = source["delivery_sequence"];
	        this.total_deliveries = source["total_deliveries"];
	        this.items = this.convertValues(source["items"], crm.DeliveryNoteItem);
	        this.customer_area = source["customer_area"];
	        this.urgency_score = source["urgency_score"];
	        this.days_until_due = source["days_until_due"];
	        this.order_value = source["order_value"];
	        this.is_high_priority = source["is_high_priority"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DeploymentDataAudit {
	    generated_at: string;
	    database_path: string;
	    expected_runtime_database_path: string;
	    packaged_database_path: string;
	    using_runtime_app_data: boolean;
	    runtime_database_exists: boolean;
	    packaged_database_exists: boolean;
	    missing_tables: string[];
	    blocking_data_issues: string[];
	    warning_data_issues: string[];
	    blocking: boolean;
	    active_customers: number;
	    active_invoices: number;
	    active_invoices_without_items: number;
	    active_orders: number;
	    active_orders_without_items: number;
	    active_zero_total_orders: number;
	    active_offers: number;
	    active_operational_offers: number;
	    active_operational_offers_without_items: number;
	    won_offers_without_items: number;
	    legacy_quoted_offer_shells: number;
	    legacy_rfq_offer_shells: number;
	    employees: number;
	    notifications: number;
	    task_items: number;
	    expense_entries: number;
	    payroll_runs: number;
	    legacy_followup_tasks: number;
	    migrated_legacy_tasks: number;
	
	    static createFrom(source: any = {}) {
	        return new DeploymentDataAudit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.generated_at = source["generated_at"];
	        this.database_path = source["database_path"];
	        this.expected_runtime_database_path = source["expected_runtime_database_path"];
	        this.packaged_database_path = source["packaged_database_path"];
	        this.using_runtime_app_data = source["using_runtime_app_data"];
	        this.runtime_database_exists = source["runtime_database_exists"];
	        this.packaged_database_exists = source["packaged_database_exists"];
	        this.missing_tables = source["missing_tables"];
	        this.blocking_data_issues = source["blocking_data_issues"];
	        this.warning_data_issues = source["warning_data_issues"];
	        this.blocking = source["blocking"];
	        this.active_customers = source["active_customers"];
	        this.active_invoices = source["active_invoices"];
	        this.active_invoices_without_items = source["active_invoices_without_items"];
	        this.active_orders = source["active_orders"];
	        this.active_orders_without_items = source["active_orders_without_items"];
	        this.active_zero_total_orders = source["active_zero_total_orders"];
	        this.active_offers = source["active_offers"];
	        this.active_operational_offers = source["active_operational_offers"];
	        this.active_operational_offers_without_items = source["active_operational_offers_without_items"];
	        this.won_offers_without_items = source["won_offers_without_items"];
	        this.legacy_quoted_offer_shells = source["legacy_quoted_offer_shells"];
	        this.legacy_rfq_offer_shells = source["legacy_rfq_offer_shells"];
	        this.employees = source["employees"];
	        this.notifications = source["notifications"];
	        this.task_items = source["task_items"];
	        this.expense_entries = source["expense_entries"];
	        this.payroll_runs = source["payroll_runs"];
	        this.legacy_followup_tasks = source["legacy_followup_tasks"];
	        this.migrated_legacy_tasks = source["migrated_legacy_tasks"];
	    }
	}
	export class DepositsInTransitResult {
	    deposits: finance.DepositInTransit[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new DepositsInTransitResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.deposits = this.convertValues(source["deposits"], finance.DepositInTransit);
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DiscoveredFile {
	    file_name: string;
	    file_path: string;
	    file_type: string;
	    extension: string;
	    size_bytes: number;
	    mod_time: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new DiscoveredFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_name = source["file_name"];
	        this.file_path = source["file_path"];
	        this.file_type = source["file_type"];
	        this.extension = source["extension"];
	        this.size_bytes = source["size_bytes"];
	        this.mod_time = this.convertValues(source["mod_time"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DiscoveredDeal {
	    local_id: string;
	    folder_path: string;
	    folder_name: string;
	    final_path: string;
	    root_path: string;
	    customer_matches: CustomerMatchResult[];
	    files: DiscoveredFile[];
	    instrument_type: string;
	    year_hint: string;
	    status: string;
	    error_msg?: string;
	    confirmed_customer_id?: string;
	    imported_offer_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new DiscoveredDeal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.local_id = source["local_id"];
	        this.folder_path = source["folder_path"];
	        this.folder_name = source["folder_name"];
	        this.final_path = source["final_path"];
	        this.root_path = source["root_path"];
	        this.customer_matches = this.convertValues(source["customer_matches"], CustomerMatchResult);
	        this.files = this.convertValues(source["files"], DiscoveredFile);
	        this.instrument_type = source["instrument_type"];
	        this.year_hint = source["year_hint"];
	        this.status = source["status"];
	        this.error_msg = source["error_msg"];
	        this.confirmed_customer_id = source["confirmed_customer_id"];
	        this.imported_offer_id = source["imported_offer_id"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class DivisionFinancialSummary {
	    division: string;
	    year: number;
	    revenue: number;
	    invoice_count: number;
	    order_count: number;
	    outstanding_ar: number;
	    paid_amount: number;
	    overdue_amount: number;
	    overdue_count: number;
	    avg_invoice_size: number;
	    source: string;
	    is_audited: boolean;
	    cost_of_sales: number;
	    gross_profit: number;
	    staff_costs: number;
	    admin_expenses: number;
	    net_profit: number;
	    cash_equivalents: number;
	    trade_receivables: number;
	    total_assets: number;
	    total_liabilities: number;
	    total_equity: number;
	    has_data: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DivisionFinancialSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.division = source["division"];
	        this.year = source["year"];
	        this.revenue = source["revenue"];
	        this.invoice_count = source["invoice_count"];
	        this.order_count = source["order_count"];
	        this.outstanding_ar = source["outstanding_ar"];
	        this.paid_amount = source["paid_amount"];
	        this.overdue_amount = source["overdue_amount"];
	        this.overdue_count = source["overdue_count"];
	        this.avg_invoice_size = source["avg_invoice_size"];
	        this.source = source["source"];
	        this.is_audited = source["is_audited"];
	        this.cost_of_sales = source["cost_of_sales"];
	        this.gross_profit = source["gross_profit"];
	        this.staff_costs = source["staff_costs"];
	        this.admin_expenses = source["admin_expenses"];
	        this.net_profit = source["net_profit"];
	        this.cash_equivalents = source["cash_equivalents"];
	        this.trade_receivables = source["trade_receivables"];
	        this.total_assets = source["total_assets"];
	        this.total_liabilities = source["total_liabilities"];
	        this.total_equity = source["total_equity"];
	        this.has_data = source["has_data"];
	    }
	}
	export class Employee {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    employee_code: string;
	    full_name: string;
	    preferred_name: string;
	    email: string;
	    phone: string;
	    department: string;
	    job_title: string;
	    employment_status: string;
	    manager_employee_id?: string;
	    start_date?: time.Time;
	    end_date?: time.Time;
	    emergency_contact: string;
	    notes: string;
	    is_active: boolean;
	    archived_at?: time.Time;
	    archived_by: string;
	    archive_reason: string;
	    archive_request_id: string;
	    manager_name?: string;
	
	    static createFrom(source: any = {}) {
	        return new Employee(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.employee_code = source["employee_code"];
	        this.full_name = source["full_name"];
	        this.preferred_name = source["preferred_name"];
	        this.email = source["email"];
	        this.phone = source["phone"];
	        this.department = source["department"];
	        this.job_title = source["job_title"];
	        this.employment_status = source["employment_status"];
	        this.manager_employee_id = source["manager_employee_id"];
	        this.start_date = this.convertValues(source["start_date"], time.Time);
	        this.end_date = this.convertValues(source["end_date"], time.Time);
	        this.emergency_contact = source["emergency_contact"];
	        this.notes = source["notes"];
	        this.is_active = source["is_active"];
	        this.archived_at = this.convertValues(source["archived_at"], time.Time);
	        this.archived_by = source["archived_by"];
	        this.archive_reason = source["archive_reason"];
	        this.archive_request_id = source["archive_request_id"];
	        this.manager_name = source["manager_name"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EmployeeAccessLink {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    employee_id: string;
	    license_key: string;
	    user_id: string;
	    device_id: string;
	    access_status: string;
	    is_primary: boolean;
	    employee_name?: string;
	    device_name?: string;
	
	    static createFrom(source: any = {}) {
	        return new EmployeeAccessLink(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.employee_id = source["employee_id"];
	        this.license_key = source["license_key"];
	        this.user_id = source["user_id"];
	        this.device_id = source["device_id"];
	        this.access_status = source["access_status"];
	        this.is_primary = source["is_primary"];
	        this.employee_name = source["employee_name"];
	        this.device_name = source["device_name"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EmployeeArchiveRequest {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    employee_id: string;
	    employee_name: string;
	    requested_by: string;
	    requested_by_name: string;
	    reason: string;
	    status: string;
	    required_approvals: number;
	    first_approved_by: string;
	    first_approved_by_name: string;
	    first_approved_at?: time.Time;
	    second_approved_by: string;
	    second_approved_by_name: string;
	    second_approved_at?: time.Time;
	    rejected_by: string;
	    rejected_by_name: string;
	    rejected_at?: time.Time;
	    review_notes: string;
	
	    static createFrom(source: any = {}) {
	        return new EmployeeArchiveRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.employee_id = source["employee_id"];
	        this.employee_name = source["employee_name"];
	        this.requested_by = source["requested_by"];
	        this.requested_by_name = source["requested_by_name"];
	        this.reason = source["reason"];
	        this.status = source["status"];
	        this.required_approvals = source["required_approvals"];
	        this.first_approved_by = source["first_approved_by"];
	        this.first_approved_by_name = source["first_approved_by_name"];
	        this.first_approved_at = this.convertValues(source["first_approved_at"], time.Time);
	        this.second_approved_by = source["second_approved_by"];
	        this.second_approved_by_name = source["second_approved_by_name"];
	        this.second_approved_at = this.convertValues(source["second_approved_at"], time.Time);
	        this.rejected_by = source["rejected_by"];
	        this.rejected_by_name = source["rejected_by_name"];
	        this.rejected_at = this.convertValues(source["rejected_at"], time.Time);
	        this.review_notes = source["review_notes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EmployeeContributionSummary {
	    employee_id: string;
	    employee_code: string;
	    employee_name: string;
	    department: string;
	    job_title: string;
	    manager_employee_id: string;
	    manager_name: string;
	    employment_status: string;
	    is_active: boolean;
	    active_project_count: number;
	    active_task_count: number;
	    completed_task_count: number;
	    blocked_task_count: number;
	    overdue_task_count: number;
	    completion_rate: number;
	    opportunity_ytd: number;
	    opportunity_won_ytd: number;
	    opportunity_lost_ytd: number;
	    revenue_ytd: number;
	    primary_license_key?: string;
	    primary_device_name?: string;
	
	    static createFrom(source: any = {}) {
	        return new EmployeeContributionSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.employee_id = source["employee_id"];
	        this.employee_code = source["employee_code"];
	        this.employee_name = source["employee_name"];
	        this.department = source["department"];
	        this.job_title = source["job_title"];
	        this.manager_employee_id = source["manager_employee_id"];
	        this.manager_name = source["manager_name"];
	        this.employment_status = source["employment_status"];
	        this.is_active = source["is_active"];
	        this.active_project_count = source["active_project_count"];
	        this.active_task_count = source["active_task_count"];
	        this.completed_task_count = source["completed_task_count"];
	        this.blocked_task_count = source["blocked_task_count"];
	        this.overdue_task_count = source["overdue_task_count"];
	        this.completion_rate = source["completion_rate"];
	        this.opportunity_ytd = source["opportunity_ytd"];
	        this.opportunity_won_ytd = source["opportunity_won_ytd"];
	        this.opportunity_lost_ytd = source["opportunity_lost_ytd"];
	        this.revenue_ytd = source["revenue_ytd"];
	        this.primary_license_key = source["primary_license_key"];
	        this.primary_device_name = source["primary_device_name"];
	    }
	}
	export class EmployeeDocument {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    employee_id: string;
	    doc_type: string;
	    permit_subtype?: string;
	    expires_on?: time.Time;
	    notes?: string;
	    notified_at?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new EmployeeDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.employee_id = source["employee_id"];
	        this.doc_type = source["doc_type"];
	        this.permit_subtype = source["permit_subtype"];
	        this.expires_on = this.convertValues(source["expires_on"], time.Time);
	        this.notes = source["notes"];
	        this.notified_at = this.convertValues(source["notified_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EmployeeDocumentDTO {
	    id: string;
	    employee_id: string;
	    doc_type: string;
	    permit_subtype?: string;
	    doc_number: string;
	    doc_number_masked: string;
	    expires_on?: time.Time;
	    notes?: string;
	    notified_at?: time.Time;
	    created_at: time.Time;
	    updated_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new EmployeeDocumentDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.employee_id = source["employee_id"];
	        this.doc_type = source["doc_type"];
	        this.permit_subtype = source["permit_subtype"];
	        this.doc_number = source["doc_number"];
	        this.doc_number_masked = source["doc_number_masked"];
	        this.expires_on = this.convertValues(source["expires_on"], time.Time);
	        this.notes = source["notes"];
	        this.notified_at = this.convertValues(source["notified_at"], time.Time);
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EmployeeDocumentScanResult {
	    scanned_at: time.Time;
	    notified_count: number;
	    lookahead_days: number;
	
	    static createFrom(source: any = {}) {
	        return new EmployeeDocumentScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scanned_at = this.convertValues(source["scanned_at"], time.Time);
	        this.notified_count = source["notified_count"];
	        this.lookahead_days = source["lookahead_days"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ExcelImportResult {
	    file_path: string;
	    file_name: string;
	    success: boolean;
	    message: string;
	    item_count: number;
	    grand_total: number;
	    customer: string;
	    offer_number: string;
	
	    static createFrom(source: any = {}) {
	        return new ExcelImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_path = source["file_path"];
	        this.file_name = source["file_name"];
	        this.success = source["success"];
	        this.message = source["message"];
	        this.item_count = source["item_count"];
	        this.grand_total = source["grand_total"];
	        this.customer = source["customer"];
	        this.offer_number = source["offer_number"];
	    }
	}
	export class ExcelBatchImportResult {
	    total_files: number;
	    successful: number;
	    failed: number;
	    results: ExcelImportResult[];
	    total_line_items: number;
	    total_value: number;
	
	    static createFrom(source: any = {}) {
	        return new ExcelBatchImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_files = source["total_files"];
	        this.successful = source["successful"];
	        this.failed = source["failed"];
	        this.results = this.convertValues(source["results"], ExcelImportResult);
	        this.total_line_items = source["total_line_items"];
	        this.total_value = source["total_value"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ExcelCostingTotals {
	    subtotal: number;
	    vat_percent: number;
	    vat_amount: number;
	    grand_total: number;
	
	    static createFrom(source: any = {}) {
	        return new ExcelCostingTotals(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.subtotal = source["subtotal"];
	        this.vat_percent = source["vat_percent"];
	        this.vat_amount = source["vat_amount"];
	        this.grand_total = source["grand_total"];
	    }
	}
	export class ExcelCostingLineItem {
	    product_number: number;
	    column_index: number;
	    supplier: string;
	    equipment: string;
	    model: string;
	    specification: string;
	    quantity: number;
	    fob_eur: number;
	    freight_eur: number;
	    exchange_rate: number;
	    fob_bhd: number;
	    freight_bhd: number;
	    cnf_bhd: number;
	    insurance: number;
	    customs: number;
	    landed_cost: number;
	    handling: number;
	    finance_charges: number;
	    other_costs: number;
	    total_cost: number;
	    markup_percent: number;
	    selling_price_bhd: number;
	    suggested_price_bhd: number;
	    total_suggested_bhd: number;
	
	    static createFrom(source: any = {}) {
	        return new ExcelCostingLineItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.product_number = source["product_number"];
	        this.column_index = source["column_index"];
	        this.supplier = source["supplier"];
	        this.equipment = source["equipment"];
	        this.model = source["model"];
	        this.specification = source["specification"];
	        this.quantity = source["quantity"];
	        this.fob_eur = source["fob_eur"];
	        this.freight_eur = source["freight_eur"];
	        this.exchange_rate = source["exchange_rate"];
	        this.fob_bhd = source["fob_bhd"];
	        this.freight_bhd = source["freight_bhd"];
	        this.cnf_bhd = source["cnf_bhd"];
	        this.insurance = source["insurance"];
	        this.customs = source["customs"];
	        this.landed_cost = source["landed_cost"];
	        this.handling = source["handling"];
	        this.finance_charges = source["finance_charges"];
	        this.other_costs = source["other_costs"];
	        this.total_cost = source["total_cost"];
	        this.markup_percent = source["markup_percent"];
	        this.selling_price_bhd = source["selling_price_bhd"];
	        this.suggested_price_bhd = source["suggested_price_bhd"];
	        this.total_suggested_bhd = source["total_suggested_bhd"];
	    }
	}
	export class ExcelCostingMetadata {
	    date: string;
	    folder_number: string;
	    est_delivery: string;
	    prepared_by: string;
	    costing_id: string;
	    delivery_terms: string;
	    customer: string;
	    contact_person: string;
	    order_type: string;
	    reference: string;
	    payment_terms: string;
	    country_of_origin: string;
	
	    static createFrom(source: any = {}) {
	        return new ExcelCostingMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.folder_number = source["folder_number"];
	        this.est_delivery = source["est_delivery"];
	        this.prepared_by = source["prepared_by"];
	        this.costing_id = source["costing_id"];
	        this.delivery_terms = source["delivery_terms"];
	        this.customer = source["customer"];
	        this.contact_person = source["contact_person"];
	        this.order_type = source["order_type"];
	        this.reference = source["reference"];
	        this.payment_terms = source["payment_terms"];
	        this.country_of_origin = source["country_of_origin"];
	    }
	}
	export class ExcelCostingData {
	    file_path: string;
	    file_name: string;
	    metadata: ExcelCostingMetadata;
	    line_items: ExcelCostingLineItem[];
	    totals: ExcelCostingTotals;
	    parsed_at: time.Time;
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new ExcelCostingData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_path = source["file_path"];
	        this.file_name = source["file_name"];
	        this.metadata = this.convertValues(source["metadata"], ExcelCostingMetadata);
	        this.line_items = this.convertValues(source["line_items"], ExcelCostingLineItem);
	        this.totals = this.convertValues(source["totals"], ExcelCostingTotals);
	        this.parsed_at = this.convertValues(source["parsed_at"], time.Time);
	        this.warnings = source["warnings"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	export class ExpenseDashboardSummary {
	    total_drafts: number;
	    total_submitted: number;
	    total_approved_unpaid: number;
	    total_recurring: number;
	    month_to_date_spend: number;
	    upcoming_commitments: number;
	
	    static createFrom(source: any = {}) {
	        return new ExpenseDashboardSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_drafts = source["total_drafts"];
	        this.total_submitted = source["total_submitted"];
	        this.total_approved_unpaid = source["total_approved_unpaid"];
	        this.total_recurring = source["total_recurring"];
	        this.month_to_date_spend = source["month_to_date_spend"];
	        this.upcoming_commitments = source["upcoming_commitments"];
	    }
	}
	export class FileSyncState {
	    Path: string;
	    EventType: string;
	    Status: string;
	    LastModified: time.Time;
	    LastSynced: time.Time;
	    RemoteHash: string;
	    LocalHash: string;
	    RetryCount: number;
	    LastError: string;
	    Metadata: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new FileSyncState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Path = source["Path"];
	        this.EventType = source["EventType"];
	        this.Status = source["Status"];
	        this.LastModified = this.convertValues(source["LastModified"], time.Time);
	        this.LastSynced = this.convertValues(source["LastSynced"], time.Time);
	        this.RemoteHash = source["RemoteHash"];
	        this.LocalHash = source["LocalHash"];
	        this.RetryCount = source["RetryCount"];
	        this.LastError = source["LastError"];
	        this.Metadata = source["Metadata"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FileWatchEvent {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    file_path: string;
	    event_type: string;
	
	    static createFrom(source: any = {}) {
	        return new FileWatchEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.file_path = source["file_path"];
	        this.event_type = source["event_type"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FilesystemDocClassification {
	    file_path: string;
	    file_name: string;
	    document_type: string;
	    offer_number: string;
	    customer_name: string;
	    product_type: string;
	    stage: string;
	    parent_folder: string;
	    file_size: number;
	    mod_time: time.Time;
	    extension: string;
	
	    static createFrom(source: any = {}) {
	        return new FilesystemDocClassification(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_path = source["file_path"];
	        this.file_name = source["file_name"];
	        this.document_type = source["document_type"];
	        this.offer_number = source["offer_number"];
	        this.customer_name = source["customer_name"];
	        this.product_type = source["product_type"];
	        this.stage = source["stage"];
	        this.parent_folder = source["parent_folder"];
	        this.file_size = source["file_size"];
	        this.mod_time = this.convertValues(source["mod_time"], time.Time);
	        this.extension = source["extension"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FilesystemClassificationSummary {
	    total_files: number;
	    by_type: Record<string, number>;
	    by_customer: Record<string, number>;
	    by_offer_number: Record<string, number>;
	    by_stage: Record<string, number>;
	    by_product_type: Record<string, number>;
	    documents: FilesystemDocClassification[];
	    scan_duration: number;
	
	    static createFrom(source: any = {}) {
	        return new FilesystemClassificationSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_files = source["total_files"];
	        this.by_type = source["by_type"];
	        this.by_customer = source["by_customer"];
	        this.by_offer_number = source["by_offer_number"];
	        this.by_stage = source["by_stage"];
	        this.by_product_type = source["by_product_type"];
	        this.documents = this.convertValues(source["documents"], FilesystemDocClassification);
	        this.scan_duration = source["scan_duration"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class FinancialDashboard {
	    period: string;
	    prior_year: string;
	    as_of_date: string;
	    source: string;
	    revenue: number;
	    revenue_yoy: number;
	    cogs: number;
	    gross_profit: number;
	    gross_margin: number;
	    opex: number;
	    ebitda: number;
	    ebitda_margin: number;
	    net_profit: number;
	    net_margin: number;
	    total_assets: number;
	    current_assets: number;
	    non_current_assets: number;
	    total_liabilities: number;
	    current_liabilities: number;
	    total_equity: number;
	    cash_and_equiv: number;
	    fixed_deposits: number;
	    total_liquidity: number;
	    trade_receivables: number;
	    inventory: number;
	    trade_payables: number;
	    working_capital: number;
	    current_ratio: number;
	    quick_ratio: number;
	    cash_ratio: number;
	    debt_to_equity: number;
	    equity_ratio: number;
	    dso: number;
	    dio: number;
	    dpo: number;
	    cash_conv_cycle: number;
	    asset_turnover: number;
	    receivables_turn: number;
	    roa: number;
	    roe: number;
	    ar_current: number;
	    ar_30_60: number;
	    ar_60_90: number;
	    ar_over_90: number;
	    ar_overdue: number;
	    ar_overdue_pct: number;
	    py_revenue: number;
	    py_gross_profit: number;
	    py_net_profit: number;
	    py_total_assets: number;
	
	    static createFrom(source: any = {}) {
	        return new FinancialDashboard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.period = source["period"];
	        this.prior_year = source["prior_year"];
	        this.as_of_date = source["as_of_date"];
	        this.source = source["source"];
	        this.revenue = source["revenue"];
	        this.revenue_yoy = source["revenue_yoy"];
	        this.cogs = source["cogs"];
	        this.gross_profit = source["gross_profit"];
	        this.gross_margin = source["gross_margin"];
	        this.opex = source["opex"];
	        this.ebitda = source["ebitda"];
	        this.ebitda_margin = source["ebitda_margin"];
	        this.net_profit = source["net_profit"];
	        this.net_margin = source["net_margin"];
	        this.total_assets = source["total_assets"];
	        this.current_assets = source["current_assets"];
	        this.non_current_assets = source["non_current_assets"];
	        this.total_liabilities = source["total_liabilities"];
	        this.current_liabilities = source["current_liabilities"];
	        this.total_equity = source["total_equity"];
	        this.cash_and_equiv = source["cash_and_equiv"];
	        this.fixed_deposits = source["fixed_deposits"];
	        this.total_liquidity = source["total_liquidity"];
	        this.trade_receivables = source["trade_receivables"];
	        this.inventory = source["inventory"];
	        this.trade_payables = source["trade_payables"];
	        this.working_capital = source["working_capital"];
	        this.current_ratio = source["current_ratio"];
	        this.quick_ratio = source["quick_ratio"];
	        this.cash_ratio = source["cash_ratio"];
	        this.debt_to_equity = source["debt_to_equity"];
	        this.equity_ratio = source["equity_ratio"];
	        this.dso = source["dso"];
	        this.dio = source["dio"];
	        this.dpo = source["dpo"];
	        this.cash_conv_cycle = source["cash_conv_cycle"];
	        this.asset_turnover = source["asset_turnover"];
	        this.receivables_turn = source["receivables_turn"];
	        this.roa = source["roa"];
	        this.roe = source["roe"];
	        this.ar_current = source["ar_current"];
	        this.ar_30_60 = source["ar_30_60"];
	        this.ar_60_90 = source["ar_60_90"];
	        this.ar_over_90 = source["ar_over_90"];
	        this.ar_overdue = source["ar_overdue"];
	        this.ar_overdue_pct = source["ar_overdue_pct"];
	        this.py_revenue = source["py_revenue"];
	        this.py_gross_profit = source["py_gross_profit"];
	        this.py_net_profit = source["py_net_profit"];
	        this.py_total_assets = source["py_total_assets"];
	    }
	}
	export class FinancialYearSummary {
	    year: number;
	    is_audited: boolean;
	    source: string;
	    as_of_date: string;
	    revenue: number;
	    cost_of_sales: number;
	    gross_profit: number;
	    other_income: number;
	    staff_costs: number;
	    admin_expenses: number;
	    depreciation: number;
	    finance_costs: number;
	    net_profit: number;
	    plant_equipment: number;
	    right_of_use: number;
	    investment_property: number;
	    inventories: number;
	    trade_receivables: number;
	    fixed_deposits: number;
	    related_party_receivable: number;
	    cash_equivalents: number;
	    total_assets: number;
	    current_assets: number;
	    non_current_assets: number;
	    share_capital: number;
	    statutory_reserve: number;
	    owner_current_account: number;
	    total_equity: number;
	    lease_non_current: number;
	    lease_current: number;
	    trade_payables: number;
	    total_liabilities: number;
	    current_liabilities: number;
	
	    static createFrom(source: any = {}) {
	        return new FinancialYearSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year = source["year"];
	        this.is_audited = source["is_audited"];
	        this.source = source["source"];
	        this.as_of_date = source["as_of_date"];
	        this.revenue = source["revenue"];
	        this.cost_of_sales = source["cost_of_sales"];
	        this.gross_profit = source["gross_profit"];
	        this.other_income = source["other_income"];
	        this.staff_costs = source["staff_costs"];
	        this.admin_expenses = source["admin_expenses"];
	        this.depreciation = source["depreciation"];
	        this.finance_costs = source["finance_costs"];
	        this.net_profit = source["net_profit"];
	        this.plant_equipment = source["plant_equipment"];
	        this.right_of_use = source["right_of_use"];
	        this.investment_property = source["investment_property"];
	        this.inventories = source["inventories"];
	        this.trade_receivables = source["trade_receivables"];
	        this.fixed_deposits = source["fixed_deposits"];
	        this.related_party_receivable = source["related_party_receivable"];
	        this.cash_equivalents = source["cash_equivalents"];
	        this.total_assets = source["total_assets"];
	        this.current_assets = source["current_assets"];
	        this.non_current_assets = source["non_current_assets"];
	        this.share_capital = source["share_capital"];
	        this.statutory_reserve = source["statutory_reserve"];
	        this.owner_current_account = source["owner_current_account"];
	        this.total_equity = source["total_equity"];
	        this.lease_non_current = source["lease_non_current"];
	        this.lease_current = source["lease_current"];
	        this.trade_payables = source["trade_payables"];
	        this.total_liabilities = source["total_liabilities"];
	        this.current_liabilities = source["current_liabilities"];
	    }
	}
	export class FolderStructureResult {
	    success: boolean;
	    created: string[];
	    inbox_path: string;
	
	    static createFrom(source: any = {}) {
	        return new FolderStructureResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.created = source["created"];
	        this.inbox_path = source["inbox_path"];
	    }
	}
	export class ItemFulfillment {
	    item_id: string;
	    line_number: number;
	    product_code: string;
	    description: string;
	    quantity: number;
	    quantity_shipped: number;
	    quantity_invoiced: number;
	    remaining_to_ship: number;
	    shipped_pct: number;
	
	    static createFrom(source: any = {}) {
	        return new ItemFulfillment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.item_id = source["item_id"];
	        this.line_number = source["line_number"];
	        this.product_code = source["product_code"];
	        this.description = source["description"];
	        this.quantity = source["quantity"];
	        this.quantity_shipped = source["quantity_shipped"];
	        this.quantity_invoiced = source["quantity_invoiced"];
	        this.remaining_to_ship = source["remaining_to_ship"];
	        this.shipped_pct = source["shipped_pct"];
	    }
	}
	export class FulfillmentStatus {
	    order_id: string;
	    total_items: number;
	    total_quantity: number;
	    shipped_quantity: number;
	    invoiced_quantity: number;
	    fulfillment_pct: number;
	    invoicing_pct: number;
	    status: string;
	    items: ItemFulfillment[];
	
	    static createFrom(source: any = {}) {
	        return new FulfillmentStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.order_id = source["order_id"];
	        this.total_items = source["total_items"];
	        this.total_quantity = source["total_quantity"];
	        this.shipped_quantity = source["shipped_quantity"];
	        this.invoiced_quantity = source["invoiced_quantity"];
	        this.fulfillment_pct = source["fulfillment_pct"];
	        this.invoicing_pct = source["invoicing_pct"];
	        this.status = source["status"];
	        this.items = this.convertValues(source["items"], ItemFulfillment);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FuzzyMatchResult {
	    extracted_name: string;
	    canonical_name: string;
	    customer_id: string;
	    match_score: number;
	    match_method: string;
	    needs_review: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FuzzyMatchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.extracted_name = source["extracted_name"];
	        this.canonical_name = source["canonical_name"];
	        this.customer_id = source["customer_id"];
	        this.match_score = source["match_score"];
	        this.match_method = source["match_method"];
	        this.needs_review = source["needs_review"];
	    }
	}
	export class GPUInfo {
	    detected: boolean;
	    vendor: string;
	    device_name: string;
	    vram_mb: number;
	    level_zero_ok: boolean;
	    cuda_ok: boolean;
	    use_gpu: boolean;
	    kernels_loaded: number;
	
	    static createFrom(source: any = {}) {
	        return new GPUInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.detected = source["detected"];
	        this.vendor = source["vendor"];
	        this.device_name = source["device_name"];
	        this.vram_mb = source["vram_mb"];
	        this.level_zero_ok = source["level_zero_ok"];
	        this.cuda_ok = source["cuda_ok"];
	        this.use_gpu = source["use_gpu"];
	        this.kernels_loaded = source["kernels_loaded"];
	    }
	}
	export class GRNItemWithSerials {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    grn_id: string;
	    po_item_id: string;
	    product_id: string;
	    quantity_ordered: number;
	    quantity_received: number;
	    quantity_accepted: number;
	    quantity_rejected: number;
	    rejection_reason: string;
	    serial_numbers: string[];
	
	    static createFrom(source: any = {}) {
	        return new GRNItemWithSerials(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.grn_id = source["grn_id"];
	        this.po_item_id = source["po_item_id"];
	        this.product_id = source["product_id"];
	        this.quantity_ordered = source["quantity_ordered"];
	        this.quantity_received = source["quantity_received"];
	        this.quantity_accepted = source["quantity_accepted"];
	        this.quantity_rejected = source["quantity_rejected"];
	        this.rejection_reason = source["rejection_reason"];
	        this.serial_numbers = source["serial_numbers"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GRNResponse {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    purchase_order_id: string;
	    grn_number: string;
	    received_date: time.Time;
	    received_by: string;
	    warehouse_id: string;
	    supplier_dn_number: string;
	    qc_status: string;
	    qc_notes: string;
	    qc_date?: time.Time;
	    qc_by: string;
	    completed_at?: time.Time;
	    updated_by: string;
	    items?: crm.GRNItem[];
	    supplier_name: string;
	    po_number: string;
	    items_count: number;
	    total_received: number;
	    total_accepted: number;
	    total_rejected: number;
	    acceptance_rate: number;
	    is_completed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GRNResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.purchase_order_id = source["purchase_order_id"];
	        this.grn_number = source["grn_number"];
	        this.received_date = this.convertValues(source["received_date"], time.Time);
	        this.received_by = source["received_by"];
	        this.warehouse_id = source["warehouse_id"];
	        this.supplier_dn_number = source["supplier_dn_number"];
	        this.qc_status = source["qc_status"];
	        this.qc_notes = source["qc_notes"];
	        this.qc_date = this.convertValues(source["qc_date"], time.Time);
	        this.qc_by = source["qc_by"];
	        this.completed_at = this.convertValues(source["completed_at"], time.Time);
	        this.updated_by = source["updated_by"];
	        this.items = this.convertValues(source["items"], crm.GRNItem);
	        this.supplier_name = source["supplier_name"];
	        this.po_number = source["po_number"];
	        this.items_count = source["items_count"];
	        this.total_received = source["total_received"];
	        this.total_accepted = source["total_accepted"];
	        this.total_rejected = source["total_rejected"];
	        this.acceptance_rate = source["acceptance_rate"];
	        this.is_completed = source["is_completed"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GradeDistItem {
	    grade: string;
	    count: number;
	    percentage: number;
	
	    static createFrom(source: any = {}) {
	        return new GradeDistItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.grade = source["grade"];
	        this.count = source["count"];
	        this.percentage = source["percentage"];
	    }
	}
	
	
	
	
	export class InboxDocument {
	    id: number;
	    document_id: string;
	    file_name: string;
	    file_path: string;
	    document_type: string;
	    status: string;
	    confidence: number;
	    extracted_data: Record<string, string>;
	    suggested_actions: string[];
	    processed_at: time.Time;
	    created_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new InboxDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.document_id = source["document_id"];
	        this.file_name = source["file_name"];
	        this.file_path = source["file_path"];
	        this.document_type = source["document_type"];
	        this.status = source["status"];
	        this.confidence = source["confidence"];
	        this.extracted_data = source["extracted_data"];
	        this.suggested_actions = source["suggested_actions"];
	        this.processed_at = this.convertValues(source["processed_at"], time.Time);
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InboxProcessResult {
	    document_id: string;
	    detected_type: string;
	    classification_confidence: number;
	    extracted_text: string;
	    entities: Record<string, string>;
	    suggested_actions: string[];
	    processed_at: time.Time;
	    needs_review: boolean;
	    ocr_confidence: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new InboxProcessResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.document_id = source["document_id"];
	        this.detected_type = source["detected_type"];
	        this.classification_confidence = source["classification_confidence"];
	        this.extracted_text = source["extracted_text"];
	        this.entities = source["entities"];
	        this.suggested_actions = source["suggested_actions"];
	        this.processed_at = this.convertValues(source["processed_at"], time.Time);
	        this.needs_review = source["needs_review"];
	        this.ocr_confidence = source["ocr_confidence"];
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InboxStats {
	    total_documents: number;
	    ready: number;
	    needs_review: number;
	    processed: number;
	    by_type: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new InboxStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_documents = source["total_documents"];
	        this.ready = source["ready"];
	        this.needs_review = source["needs_review"];
	        this.processed = source["processed"];
	        this.by_type = source["by_type"];
	    }
	}
	export class InitialScanResult {
	    total_files: number;
	    files_by_type: Record<string, number>;
	    conflicts: string[];
	    warnings: string[];
	    scan_duration_ms: number;
	    report_path: string;
	
	    static createFrom(source: any = {}) {
	        return new InitialScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_files = source["total_files"];
	        this.files_by_type = source["files_by_type"];
	        this.conflicts = source["conflicts"];
	        this.warnings = source["warnings"];
	        this.scan_duration_ms = source["scan_duration_ms"];
	        this.report_path = source["report_path"];
	    }
	}
	export class InventoryAlert {
	    product_id: string;
	    product_code: string;
	    product_name: string;
	    quantity_on_hand: number;
	    reorder_point: number;
	    minimum_stock: number;
	    alert_type: string;
	    alert_severity: string;
	    days_until_stock: number;
	    last_movement_at?: time.Time;
	    days_since_movement: number;
	    supplier_id: string;
	    supplier_name: string;
	    lead_time_days: number;
	
	    static createFrom(source: any = {}) {
	        return new InventoryAlert(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.product_id = source["product_id"];
	        this.product_code = source["product_code"];
	        this.product_name = source["product_name"];
	        this.quantity_on_hand = source["quantity_on_hand"];
	        this.reorder_point = source["reorder_point"];
	        this.minimum_stock = source["minimum_stock"];
	        this.alert_type = source["alert_type"];
	        this.alert_severity = source["alert_severity"];
	        this.days_until_stock = source["days_until_stock"];
	        this.last_movement_at = this.convertValues(source["last_movement_at"], time.Time);
	        this.days_since_movement = source["days_since_movement"];
	        this.supplier_id = source["supplier_id"];
	        this.supplier_name = source["supplier_name"];
	        this.lead_time_days = source["lead_time_days"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InventoryPendingFulfillmentRow {
	    order_id: string;
	    order_number: string;
	    customer_name: string;
	    product_code: string;
	    description: string;
	    ordered_quantity: number;
	    delivered_quantity: number;
	    invoiced_quantity: number;
	    pending_quantity: number;
	    available_quantity: number;
	    shortage_quantity: number;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new InventoryPendingFulfillmentRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.order_id = source["order_id"];
	        this.order_number = source["order_number"];
	        this.customer_name = source["customer_name"];
	        this.product_code = source["product_code"];
	        this.description = source["description"];
	        this.ordered_quantity = source["ordered_quantity"];
	        this.delivered_quantity = source["delivered_quantity"];
	        this.invoiced_quantity = source["invoiced_quantity"];
	        this.pending_quantity = source["pending_quantity"];
	        this.available_quantity = source["available_quantity"];
	        this.shortage_quantity = source["shortage_quantity"];
	        this.status = source["status"];
	    }
	}
	export class InvoiceAuditTrail {
	    invoice: finance.Invoice;
	    rfq?: crm.Offer;
	    quote?: crm.Offer;
	    order?: crm.Order;
	    purchase_orders: crm.PurchaseOrder[];
	    supplier_invoices: finance.SupplierInvoice[];
	    delivery_notes: crm.DeliveryNote[];
	
	    static createFrom(source: any = {}) {
	        return new InvoiceAuditTrail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.invoice = this.convertValues(source["invoice"], finance.Invoice);
	        this.rfq = this.convertValues(source["rfq"], crm.Offer);
	        this.quote = this.convertValues(source["quote"], crm.Offer);
	        this.order = this.convertValues(source["order"], crm.Order);
	        this.purchase_orders = this.convertValues(source["purchase_orders"], crm.PurchaseOrder);
	        this.supplier_invoices = this.convertValues(source["supplier_invoices"], finance.SupplierInvoice);
	        this.delivery_notes = this.convertValues(source["delivery_notes"], crm.DeliveryNote);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InvoiceHashVerification {
	    invoice_id: string;
	    invoice_number: string;
	    stored_hash: string;
	    computed_hash: string;
	    has_hash: boolean;
	    valid: boolean;
	
	    static createFrom(source: any = {}) {
	        return new InvoiceHashVerification(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.invoice_id = source["invoice_id"];
	        this.invoice_number = source["invoice_number"];
	        this.stored_hash = source["stored_hash"];
	        this.computed_hash = source["computed_hash"];
	        this.has_hash = source["has_hash"];
	        this.valid = source["valid"];
	    }
	}
	
	
	export class ReportGenerateOutput {
	    file_path: string;
	    file_size: number;
	    row_count?: number;
	    generated: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new ReportGenerateOutput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_path = source["file_path"];
	        this.file_size = source["file_size"];
	        this.row_count = source["row_count"];
	        this.generated = this.convertValues(source["generated"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class JobStatusResponse {
	    id: string;
	    type: string;
	    status: string;
	    progress: number;
	    error?: string;
	    output?: ReportGenerateOutput;
	    started_at?: any;
	    completed_at?: any;
	
	    static createFrom(source: any = {}) {
	        return new JobStatusResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.progress = source["progress"];
	        this.error = source["error"];
	        this.output = this.convertValues(source["output"], ReportGenerateOutput);
	        this.started_at = source["started_at"];
	        this.completed_at = source["completed_at"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class JobSummary {
	    id: string;
	    type: string;
	    status: string;
	    progress: number;
	    created_at: any;
	    completed_at?: any;
	
	    static createFrom(source: any = {}) {
	        return new JobSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.progress = source["progress"];
	        this.created_at = source["created_at"];
	        this.completed_at = source["completed_at"];
	    }
	}
	export class LicenseActivationResult {
	    success: boolean;
	    message: string;
	    role: string;
	    display_name: string;
	    permissions: string[];
	    device_hash: string;
	
	    static createFrom(source: any = {}) {
	        return new LicenseActivationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.role = source["role"];
	        this.display_name = source["display_name"];
	        this.permissions = source["permissions"];
	        this.device_hash = source["device_hash"];
	    }
	}
	export class LicenseKey {
	    id: number;
	    key: string;
	    role: string;
	    display_name: string;
	    device_hash: string;
	    activated: boolean;
	    activated_at?: time.Time;
	    issued_at: time.Time;
	    expires_at?: time.Time;
	    notes: string;
	    created_by: string;
	
	    static createFrom(source: any = {}) {
	        return new LicenseKey(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.key = source["key"];
	        this.role = source["role"];
	        this.display_name = source["display_name"];
	        this.device_hash = source["device_hash"];
	        this.activated = source["activated"];
	        this.activated_at = this.convertValues(source["activated_at"], time.Time);
	        this.issued_at = this.convertValues(source["issued_at"], time.Time);
	        this.expires_at = this.convertValues(source["expires_at"], time.Time);
	        this.notes = source["notes"];
	        this.created_by = source["created_by"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LicenseValidationResult {
	    valid: boolean;
	    role: string;
	    key: string;
	    display_name: string;
	    permissions: string[];
	    expires_at?: string;
	
	    static createFrom(source: any = {}) {
	        return new LicenseValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.role = source["role"];
	        this.key = source["key"];
	        this.display_name = source["display_name"];
	        this.permissions = source["permissions"];
	        this.expires_at = source["expires_at"];
	    }
	}
	export class MarginAlert {
	    severity: string;
	    message: string;
	    margin_percent: number;
	    threshold: number;
	    recommendation: string;
	
	    static createFrom(source: any = {}) {
	        return new MarginAlert(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.severity = source["severity"];
	        this.message = source["message"];
	        this.margin_percent = source["margin_percent"];
	        this.threshold = source["threshold"];
	        this.recommendation = source["recommendation"];
	    }
	}
	export class MarginAnalysisByCustomer {
	    customer_id: string;
	    customer_name: string;
	    total_revenue: number;
	    total_cost: number;
	    gross_margin: number;
	    margin_percent: number;
	    order_count: number;
	    avg_margin_pct: number;
	
	    static createFrom(source: any = {}) {
	        return new MarginAnalysisByCustomer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.total_revenue = source["total_revenue"];
	        this.total_cost = source["total_cost"];
	        this.gross_margin = source["gross_margin"];
	        this.margin_percent = source["margin_percent"];
	        this.order_count = source["order_count"];
	        this.avg_margin_pct = source["avg_margin_pct"];
	    }
	}
	export class MarginAnalysisByProduct {
	    product_category: string;
	    total_revenue: number;
	    total_cost: number;
	    gross_margin: number;
	    margin_percent: number;
	    order_count: number;
	
	    static createFrom(source: any = {}) {
	        return new MarginAnalysisByProduct(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.product_category = source["product_category"];
	        this.total_revenue = source["total_revenue"];
	        this.total_cost = source["total_cost"];
	        this.gross_margin = source["gross_margin"];
	        this.margin_percent = source["margin_percent"];
	        this.order_count = source["order_count"];
	    }
	}
	export class MarginSimulation {
	    customer: string;
	    proposed_margin: number;
	    current_win_rate: number;
	    estimated_win_rate: number;
	    confidence: number;
	    recommended_action: string;
	    warning?: string;
	
	    static createFrom(source: any = {}) {
	        return new MarginSimulation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.customer = source["customer"];
	        this.proposed_margin = source["proposed_margin"];
	        this.current_win_rate = source["current_win_rate"];
	        this.estimated_win_rate = source["estimated_win_rate"];
	        this.confidence = source["confidence"];
	        this.recommended_action = source["recommended_action"];
	        this.warning = source["warning"];
	    }
	}
	export class MasterDataDuplicateMember {
	    id: string;
	    name: string;
	    created_at: time.Time;
	    invoice_count: number;
	    order_count: number;
	    offer_count: number;
	    contact_count: number;
	    opportunity_count: number;
	    purchase_order_count: number;
	    supplier_invoice_count: number;
	    supplier_payment_count: number;
	    product_count: number;
	    activity_score: number;
	
	    static createFrom(source: any = {}) {
	        return new MasterDataDuplicateMember(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.invoice_count = source["invoice_count"];
	        this.order_count = source["order_count"];
	        this.offer_count = source["offer_count"];
	        this.contact_count = source["contact_count"];
	        this.opportunity_count = source["opportunity_count"];
	        this.purchase_order_count = source["purchase_order_count"];
	        this.supplier_invoice_count = source["supplier_invoice_count"];
	        this.supplier_payment_count = source["supplier_payment_count"];
	        this.product_count = source["product_count"];
	        this.activity_score = source["activity_score"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MasterDataDuplicateCandidate {
	    entity_type: string;
	    normalized_name: string;
	    primary_id: string;
	    primary_name: string;
	    auto_merge_safe: boolean;
	    auto_merge_reason: string;
	    members: MasterDataDuplicateMember[];
	
	    static createFrom(source: any = {}) {
	        return new MasterDataDuplicateCandidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entity_type = source["entity_type"];
	        this.normalized_name = source["normalized_name"];
	        this.primary_id = source["primary_id"];
	        this.primary_name = source["primary_name"];
	        this.auto_merge_safe = source["auto_merge_safe"];
	        this.auto_merge_reason = source["auto_merge_reason"];
	        this.members = this.convertValues(source["members"], MasterDataDuplicateMember);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MasterDataCleanupAudit {
	    generated_at: time.Time;
	    customer_candidates: MasterDataDuplicateCandidate[];
	    supplier_candidates: MasterDataDuplicateCandidate[];
	    auto_merge_customer_ids: string[];
	    auto_merge_supplier_ids: string[];
	
	    static createFrom(source: any = {}) {
	        return new MasterDataCleanupAudit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.generated_at = this.convertValues(source["generated_at"], time.Time);
	        this.customer_candidates = this.convertValues(source["customer_candidates"], MasterDataDuplicateCandidate);
	        this.supplier_candidates = this.convertValues(source["supplier_candidates"], MasterDataDuplicateCandidate);
	        this.auto_merge_customer_ids = source["auto_merge_customer_ids"];
	        this.auto_merge_supplier_ids = source["auto_merge_supplier_ids"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class MonthlyPLBreakdown {
	    month: number;
	    month_name: string;
	    revenue: number;
	    purchases: number;
	    gross_profit: number;
	    net_profit: number;
	
	    static createFrom(source: any = {}) {
	        return new MonthlyPLBreakdown(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.month = source["month"];
	        this.month_name = source["month_name"];
	        this.revenue = source["revenue"];
	        this.purchases = source["purchases"];
	        this.gross_profit = source["gross_profit"];
	        this.net_profit = source["net_profit"];
	    }
	}
	export class Notification {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    employee_id: string;
	    notification_type: string;
	    title: string;
	    message: string;
	    status: string;
	    source_type: string;
	    source_id: string;
	    action_route: string;
	    action_payload: string;
	    read_at?: time.Time;
	    delivered_at?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new Notification(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.employee_id = source["employee_id"];
	        this.notification_type = source["notification_type"];
	        this.title = source["title"];
	        this.message = source["message"];
	        this.status = source["status"];
	        this.source_type = source["source_type"];
	        this.source_id = source["source_id"];
	        this.action_route = source["action_route"];
	        this.action_payload = source["action_payload"];
	        this.read_at = this.convertValues(source["read_at"], time.Time);
	        this.delivered_at = this.convertValues(source["delivered_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OCRDocument {
	    id: number;
	    file_name: string;
	    file_path: string;
	    document_type: string;
	    extracted_text: string;
	    extracted_data_json: string;
	    confidence: number;
	    processing_time_ms: number;
	    engine: string;
	    tier_used: string;
	    cost: number;
	    dna_cache_hit: boolean;
	    table_detected: boolean;
	    gpu_used: boolean;
	    processed_at: time.Time;
	    created_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new OCRDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.file_name = source["file_name"];
	        this.file_path = source["file_path"];
	        this.document_type = source["document_type"];
	        this.extracted_text = source["extracted_text"];
	        this.extracted_data_json = source["extracted_data_json"];
	        this.confidence = source["confidence"];
	        this.processing_time_ms = source["processing_time_ms"];
	        this.engine = source["engine"];
	        this.tier_used = source["tier_used"];
	        this.cost = source["cost"];
	        this.dna_cache_hit = source["dna_cache_hit"];
	        this.table_detected = source["table_detected"];
	        this.gpu_used = source["gpu_used"];
	        this.processed_at = this.convertValues(source["processed_at"], time.Time);
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OCRResultSimple {
	    success: boolean;
	    text: string;
	    confidence: number;
	    document_type: string;
	    extracted_data: Record<string, any>;
	    extracted_fields: Record<string, any>;
	    processing_time_ms: number;
	    processing_time_ms_legacy: number;
	    engine: string;
	    tier_used: string;
	    cost: number;
	    dna_cache_hit: boolean;
	    table_detected: boolean;
	    gpu_used: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new OCRResultSimple(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.text = source["text"];
	        this.confidence = source["confidence"];
	        this.document_type = source["document_type"];
	        this.extracted_data = source["extracted_data"];
	        this.extracted_fields = source["extracted_fields"];
	        this.processing_time_ms = source["processing_time_ms"];
	        this.processing_time_ms_legacy = source["processing_time_ms_legacy"];
	        this.engine = source["engine"];
	        this.tier_used = source["tier_used"];
	        this.cost = source["cost"];
	        this.dna_cache_hit = source["dna_cache_hit"];
	        this.table_detected = source["table_detected"];
	        this.gpu_used = source["gpu_used"];
	        this.error = source["error"];
	    }
	}
	export class OfferData {
	    id: string;
	    costing_id: string;
	    customer_name: string;
	    project_name: string;
	    amount: number;
	    status: string;
	    pdf_path: string;
	    sent_at: time.Time;
	    created_at: time.Time;
	    updated_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new OfferData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.costing_id = source["costing_id"];
	        this.customer_name = source["customer_name"];
	        this.project_name = source["project_name"];
	        this.amount = source["amount"];
	        this.status = source["status"];
	        this.pdf_path = source["pdf_path"];
	        this.sent_at = this.convertValues(source["sent_at"], time.Time);
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OfferRevision {
	    id: string;
	    offer_id: string;
	    revision_number: number;
	    revised_by: string;
	    revision_date: time.Time;
	    revision_notes: string;
	    total_value_bhd: number;
	    stage: string;
	    items_json: string;
	
	    static createFrom(source: any = {}) {
	        return new OfferRevision(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.offer_id = source["offer_id"];
	        this.revision_number = source["revision_number"];
	        this.revised_by = source["revised_by"];
	        this.revision_date = this.convertValues(source["revision_date"], time.Time);
	        this.revision_notes = source["revision_notes"];
	        this.total_value_bhd = source["total_value_bhd"];
	        this.stage = source["stage"];
	        this.items_json = source["items_json"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OfferSearchResult {
	    id: string;
	    offer_number: string;
	    customer_id: string;
	    customer_name: string;
	    quotation_date: time.Time;
	    total_value_bhd: number;
	    stage: string;
	    revision_number: number;
	    match_score: number;
	
	    static createFrom(source: any = {}) {
	        return new OfferSearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.offer_number = source["offer_number"];
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.quotation_date = this.convertValues(source["quotation_date"], time.Time);
	        this.total_value_bhd = source["total_value_bhd"];
	        this.stage = source["stage"];
	        this.revision_number = source["revision_number"];
	        this.match_score = source["match_score"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OfferUpdateItem {
	    description: string;
	    model: string;
	    supplier: string;
	    quantity: number;
	    unit_price: number;
	    equipment: string;
	    product_code: string;
	    specification: string;
	    detailed_description: string;
	    currency: string;
	    fob: number;
	    freight: number;
	    total_cost: number;
	    margin_percent: number;
	    total_price: number;
	    exchange_rate: number;
	
	    static createFrom(source: any = {}) {
	        return new OfferUpdateItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.description = source["description"];
	        this.model = source["model"];
	        this.supplier = source["supplier"];
	        this.quantity = source["quantity"];
	        this.unit_price = source["unit_price"];
	        this.equipment = source["equipment"];
	        this.product_code = source["product_code"];
	        this.specification = source["specification"];
	        this.detailed_description = source["detailed_description"];
	        this.currency = source["currency"];
	        this.fob = source["fob"];
	        this.freight = source["freight"];
	        this.total_cost = source["total_cost"];
	        this.margin_percent = source["margin_percent"];
	        this.total_price = source["total_price"];
	        this.exchange_rate = source["exchange_rate"];
	    }
	}
	export class OfferUpdateData {
	    offer_number: string;
	    customer_id: string;
	    customer_name: string;
	    division: string;
	    project_name: string;
	    folder_number: string;
	    quotation_date: string;
	    validity_date: string;
	    payment_terms: string;
	    delivery_terms: string;
	    delivery_weeks: string;
	    country_of_origin: string;
	    issued_by: string;
	    contact_phone: string;
	    customer_reference: string;
	    attention_person: string;
	    attention_company: string;
	    attention_phone: string;
	    attention_address: string;
	    subject: string;
	    body: string;
	    quote_type: string;
	    vat_rate: number;
	    discount: number;
	    stage: string;
	    items: OfferUpdateItem[];
	
	    static createFrom(source: any = {}) {
	        return new OfferUpdateData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.offer_number = source["offer_number"];
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.division = source["division"];
	        this.project_name = source["project_name"];
	        this.folder_number = source["folder_number"];
	        this.quotation_date = source["quotation_date"];
	        this.validity_date = source["validity_date"];
	        this.payment_terms = source["payment_terms"];
	        this.delivery_terms = source["delivery_terms"];
	        this.delivery_weeks = source["delivery_weeks"];
	        this.country_of_origin = source["country_of_origin"];
	        this.issued_by = source["issued_by"];
	        this.contact_phone = source["contact_phone"];
	        this.customer_reference = source["customer_reference"];
	        this.attention_person = source["attention_person"];
	        this.attention_company = source["attention_company"];
	        this.attention_phone = source["attention_phone"];
	        this.attention_address = source["attention_address"];
	        this.subject = source["subject"];
	        this.body = source["body"];
	        this.quote_type = source["quote_type"];
	        this.vat_rate = source["vat_rate"];
	        this.discount = source["discount"];
	        this.stage = source["stage"];
	        this.items = this.convertValues(source["items"], OfferUpdateItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class OfficeInfo {
	    outlook_enabled: boolean;
	    excel_enabled: boolean;
	    word_enabled: boolean;
	    powerpoint_enabled: boolean;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new OfficeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.outlook_enabled = source["outlook_enabled"];
	        this.excel_enabled = source["excel_enabled"];
	        this.word_enabled = source["word_enabled"];
	        this.powerpoint_enabled = source["powerpoint_enabled"];
	        this.version = source["version"];
	    }
	}
	export class OneDriveImportResult {
	    deal_local_id: string;
	    success: boolean;
	    offer_id?: string;
	    message: string;
	    costing_sheets_imported: number;
	    pdfs_queued: number;
	
	    static createFrom(source: any = {}) {
	        return new OneDriveImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.deal_local_id = source["deal_local_id"];
	        this.success = source["success"];
	        this.offer_id = source["offer_id"];
	        this.message = source["message"];
	        this.costing_sheets_imported = source["costing_sheets_imported"];
	        this.pdfs_queued = source["pdfs_queued"];
	    }
	}
	export class OneDriveScanResult {
	    deals: DiscoveredDeal[];
	    total_folders: number;
	    total_files: number;
	    scan_paths: string[];
	    scanned_at: time.Time;
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new OneDriveScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.deals = this.convertValues(source["deals"], DiscoveredDeal);
	        this.total_folders = source["total_folders"];
	        this.total_files = source["total_files"];
	        this.scan_paths = source["scan_paths"];
	        this.scanned_at = this.convertValues(source["scanned_at"], time.Time);
	        this.errors = source["errors"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OperationalBalanceSheetReport {
	    as_of_date: string;
	    division: string;
	    cash_bhd: number;
	    accounts_receivable_bhd: number;
	    customer_credits_bhd: number;
	    accounts_payable_bhd: number;
	    expense_liability_bhd: number;
	    net_position_bhd: number;
	
	    static createFrom(source: any = {}) {
	        return new OperationalBalanceSheetReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.as_of_date = source["as_of_date"];
	        this.division = source["division"];
	        this.cash_bhd = source["cash_bhd"];
	        this.accounts_receivable_bhd = source["accounts_receivable_bhd"];
	        this.customer_credits_bhd = source["customer_credits_bhd"];
	        this.accounts_payable_bhd = source["accounts_payable_bhd"];
	        this.expense_liability_bhd = source["expense_liability_bhd"];
	        this.net_position_bhd = source["net_position_bhd"];
	    }
	}
	export class OperationalLedgerEntry {
	    id: string;
	    source_type: string;
	    source_id: string;
	    source_number: string;
	    entry_date: string;
	    party_id: string;
	    party_name: string;
	    party_type: string;
	    description: string;
	    debit_bhd: number;
	    credit_bhd: number;
	    balance_bhd: number;
	    status: string;
	    division: string;
	
	    static createFrom(source: any = {}) {
	        return new OperationalLedgerEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source_type = source["source_type"];
	        this.source_id = source["source_id"];
	        this.source_number = source["source_number"];
	        this.entry_date = source["entry_date"];
	        this.party_id = source["party_id"];
	        this.party_name = source["party_name"];
	        this.party_type = source["party_type"];
	        this.description = source["description"];
	        this.debit_bhd = source["debit_bhd"];
	        this.credit_bhd = source["credit_bhd"];
	        this.balance_bhd = source["balance_bhd"];
	        this.status = source["status"];
	        this.division = source["division"];
	    }
	}
	export class OperationalLedgerFilter {
	    division: string;
	    start_date: string;
	    end_date: string;
	    customer_id: string;
	    supplier_id: string;
	    vendor_id: string;
	    source_type: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new OperationalLedgerFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.division = source["division"];
	        this.start_date = source["start_date"];
	        this.end_date = source["end_date"];
	        this.customer_id = source["customer_id"];
	        this.supplier_id = source["supplier_id"];
	        this.vendor_id = source["vendor_id"];
	        this.source_type = source["source_type"];
	        this.status = source["status"];
	    }
	}
	export class OperationalProfitLossReport {
	    start_date: string;
	    end_date: string;
	    division: string;
	    revenue_bhd: number;
	    cogs_bhd: number;
	    gross_profit_bhd: number;
	    gross_margin_percent: number;
	    expenses_bhd: number;
	    net_income_bhd: number;
	    net_margin_percent: number;
	    invoice_count: number;
	    expense_count: number;
	    expense_breakdown: AccountBalance[];
	
	    static createFrom(source: any = {}) {
	        return new OperationalProfitLossReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start_date = source["start_date"];
	        this.end_date = source["end_date"];
	        this.division = source["division"];
	        this.revenue_bhd = source["revenue_bhd"];
	        this.cogs_bhd = source["cogs_bhd"];
	        this.gross_profit_bhd = source["gross_profit_bhd"];
	        this.gross_margin_percent = source["gross_margin_percent"];
	        this.expenses_bhd = source["expenses_bhd"];
	        this.net_income_bhd = source["net_income_bhd"];
	        this.net_margin_percent = source["net_margin_percent"];
	        this.invoice_count = source["invoice_count"];
	        this.expense_count = source["expense_count"];
	        this.expense_breakdown = this.convertValues(source["expense_breakdown"], AccountBalance);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OpportunityComment {
	    id: number;
	    opportunity_id: string;
	    comment: string;
	    created_by: string;
	    created_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new OpportunityComment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.opportunity_id = source["opportunity_id"];
	        this.comment = source["comment"];
	        this.created_by = source["created_by"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OpportunityEditConflict {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    opportunity_id: string;
	    folder_number: string;
	    operation: string;
	    status: string;
	    expected_version: number;
	    current_version: number;
	    attempted_by: string;
	    attempted_role: string;
	    proposed_changes_json: string;
	    current_snapshot_json: string;
	    base_snapshot_json: string;
	    resolution_action: string;
	    resolution_note: string;
	    resolved_by: string;
	    resolved_at?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new OpportunityEditConflict(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.opportunity_id = source["opportunity_id"];
	        this.folder_number = source["folder_number"];
	        this.operation = source["operation"];
	        this.status = source["status"];
	        this.expected_version = source["expected_version"];
	        this.current_version = source["current_version"];
	        this.attempted_by = source["attempted_by"];
	        this.attempted_role = source["attempted_role"];
	        this.proposed_changes_json = source["proposed_changes_json"];
	        this.current_snapshot_json = source["current_snapshot_json"];
	        this.base_snapshot_json = source["base_snapshot_json"];
	        this.resolution_action = source["resolution_action"];
	        this.resolution_note = source["resolution_note"];
	        this.resolved_by = source["resolved_by"];
	        this.resolved_at = this.convertValues(source["resolved_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OpportunityConflictResolutionResult {
	    conflict: OpportunityEditConflict;
	    opportunity: crm.Opportunity;
	
	    static createFrom(source: any = {}) {
	        return new OpportunityConflictResolutionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conflict = this.convertValues(source["conflict"], OpportunityEditConflict);
	        this.opportunity = this.convertValues(source["opportunity"], crm.Opportunity);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OpportunityDueData {
	    id: string;
	    folder_number: string;
	    customer_id: string;
	    customer_name: string;
	    expected_date?: time.Time;
	    stage: string;
	    days_overdue: number;
	    primary_contact: string;
	    primary_email: string;
	    primary_phone: string;
	    estimated_value: number;
	
	    static createFrom(source: any = {}) {
	        return new OpportunityDueData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.folder_number = source["folder_number"];
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.expected_date = this.convertValues(source["expected_date"], time.Time);
	        this.stage = source["stage"];
	        this.days_overdue = source["days_overdue"];
	        this.primary_contact = source["primary_contact"];
	        this.primary_email = source["primary_email"];
	        this.primary_phone = source["primary_phone"];
	        this.estimated_value = source["estimated_value"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class OrderFulfillmentItem {
	    order_item_id: string;
	    product_id: string;
	    product_code: string;
	    description: string;
	    ordered_qty: number;
	    shipped_qty: number;
	    delivered_qty: number;
	    invoiced_qty: number;
	    remaining_qty: number;
	    requires_serial_tracking: boolean;
	
	    static createFrom(source: any = {}) {
	        return new OrderFulfillmentItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.order_item_id = source["order_item_id"];
	        this.product_id = source["product_id"];
	        this.product_code = source["product_code"];
	        this.description = source["description"];
	        this.ordered_qty = source["ordered_qty"];
	        this.shipped_qty = source["shipped_qty"];
	        this.delivered_qty = source["delivered_qty"];
	        this.invoiced_qty = source["invoiced_qty"];
	        this.remaining_qty = source["remaining_qty"];
	        this.requires_serial_tracking = source["requires_serial_tracking"];
	    }
	}
	export class OrderFulfillment {
	    order_id: string;
	    order_number: string;
	    customer_name: string;
	    status: string;
	    items: OrderFulfillmentItem[];
	    fully_delivered: boolean;
	    fully_invoiced: boolean;
	
	    static createFrom(source: any = {}) {
	        return new OrderFulfillment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.order_id = source["order_id"];
	        this.order_number = source["order_number"];
	        this.customer_name = source["customer_name"];
	        this.status = source["status"];
	        this.items = this.convertValues(source["items"], OrderFulfillmentItem);
	        this.fully_delivered = source["fully_delivered"];
	        this.fully_invoiced = source["fully_invoiced"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class OrderSearchResult {
	    id: string;
	    order_number: string;
	    customer_po_number: string;
	    customer_id: string;
	    customer_name: string;
	    order_date: time.Time;
	    total_value_bhd: number;
	    status: string;
	    match_score: number;
	
	    static createFrom(source: any = {}) {
	        return new OrderSearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.order_number = source["order_number"];
	        this.customer_po_number = source["customer_po_number"];
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.order_date = this.convertValues(source["order_date"], time.Time);
	        this.total_value_bhd = source["total_value_bhd"];
	        this.status = source["status"];
	        this.match_score = source["match_score"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OrderStage {
	    stage: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new OrderStage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stage = source["stage"];
	        this.count = source["count"];
	    }
	}
	
	export class OverdueBucket {
	    days: string;
	    amount: number;
	
	    static createFrom(source: any = {}) {
	        return new OverdueBucket(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.days = source["days"];
	        this.amount = source["amount"];
	    }
	}
	export class POAmendment {
	    id: string;
	    purchase_order_id: string;
	    amendment_number: number;
	    amended_by: string;
	    amended_at: time.Time;
	    change_type: string;
	    old_value: string;
	    new_value: string;
	    reason: string;
	    requires_reapproval: boolean;
	    reapproved_by: string;
	    reapproved_at?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new POAmendment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.purchase_order_id = source["purchase_order_id"];
	        this.amendment_number = source["amendment_number"];
	        this.amended_by = source["amended_by"];
	        this.amended_at = this.convertValues(source["amended_at"], time.Time);
	        this.change_type = source["change_type"];
	        this.old_value = source["old_value"];
	        this.new_value = source["new_value"];
	        this.reason = source["reason"];
	        this.requires_reapproval = source["requires_reapproval"];
	        this.reapproved_by = source["reapproved_by"];
	        this.reapproved_at = this.convertValues(source["reapproved_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class POSummary {
	    id: string;
	    po_number: string;
	    po_date: time.Time;
	    total_bhd: number;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new POSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.po_number = source["po_number"];
	        this.po_date = this.convertValues(source["po_date"], time.Time);
	        this.total_bhd = source["total_bhd"];
	        this.status = source["status"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PaginationResult {
	    data: any;
	    total: number;
	    page: number;
	    page_size: number;
	    total_pages: number;
	    has_next: boolean;
	    has_prev: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PaginationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = source["data"];
	        this.total = source["total"];
	        this.page = source["page"];
	        this.page_size = source["page_size"];
	        this.total_pages = source["total_pages"];
	        this.has_next = source["has_next"];
	        this.has_prev = source["has_prev"];
	    }
	}
	
	export class PaymentAgingBucket {
	    customer_id: string;
	    customer_name: string;
	    current: number;
	    days_1_to_30: number;
	    days_31_to_60: number;
	    days_61_to_90: number;
	    over_90_days: number;
	    total_outstanding: number;
	    avg_days_overdue: number;
	
	    static createFrom(source: any = {}) {
	        return new PaymentAgingBucket(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.current = source["current"];
	        this.days_1_to_30 = source["days_1_to_30"];
	        this.days_31_to_60 = source["days_31_to_60"];
	        this.days_61_to_90 = source["days_61_to_90"];
	        this.over_90_days = source["over_90_days"];
	        this.total_outstanding = source["total_outstanding"];
	        this.avg_days_overdue = source["avg_days_overdue"];
	    }
	}
	export class PaymentAgingReport {
	    report_date: time.Time;
	    total_current: number;
	    total_days_1_to_30: number;
	    total_days_31_to_60: number;
	    total_days_61_to_90: number;
	    total_over_90_days: number;
	    grand_total: number;
	    avg_days_overdue: number;
	    customer_buckets: PaymentAgingBucket[];
	
	    static createFrom(source: any = {}) {
	        return new PaymentAgingReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.report_date = this.convertValues(source["report_date"], time.Time);
	        this.total_current = source["total_current"];
	        this.total_days_1_to_30 = source["total_days_1_to_30"];
	        this.total_days_31_to_60 = source["total_days_31_to_60"];
	        this.total_days_61_to_90 = source["total_days_61_to_90"];
	        this.total_over_90_days = source["total_over_90_days"];
	        this.grand_total = source["grand_total"];
	        this.avg_days_overdue = source["avg_days_overdue"];
	        this.customer_buckets = this.convertValues(source["customer_buckets"], PaymentAgingBucket);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Phase7ActionResult {
	    status: string;
	    message: string;
	    processed: number;
	
	    static createFrom(source: any = {}) {
	        return new Phase7ActionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.processed = source["processed"];
	    }
	}
	export class Phase7RolloutStatus {
	    followup_backfill_completed_at: string;
	    followup_backfill_count: number;
	    legacy_followup_tasks: number;
	    migrated_legacy_tasks: number;
	    pending_collaborative_ops: number;
	    failed_collaborative_ops: number;
	    dead_letter_collaborative_ops: number;
	    payroll_payouts_awaiting_recon: number;
	
	    static createFrom(source: any = {}) {
	        return new Phase7RolloutStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.followup_backfill_completed_at = source["followup_backfill_completed_at"];
	        this.followup_backfill_count = source["followup_backfill_count"];
	        this.legacy_followup_tasks = source["legacy_followup_tasks"];
	        this.migrated_legacy_tasks = source["migrated_legacy_tasks"];
	        this.pending_collaborative_ops = source["pending_collaborative_ops"];
	        this.failed_collaborative_ops = source["failed_collaborative_ops"];
	        this.dead_letter_collaborative_ops = source["dead_letter_collaborative_ops"];
	        this.payroll_payouts_awaiting_recon = source["payroll_payouts_awaiting_recon"];
	    }
	}
	export class PilotChecklistItem {
	    id: string;
	    title: string;
	    description: string;
	    completed: boolean;
	    notes: string;
	    completed_at: string;
	
	    static createFrom(source: any = {}) {
	        return new PilotChecklistItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.completed = source["completed"];
	        this.notes = source["notes"];
	        this.completed_at = source["completed_at"];
	    }
	}
	export class PilotExportResult {
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new PilotExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	    }
	}
	export class PilotReadinessRow {
	    employee_id: string;
	    employee_code: string;
	    employee_name: string;
	    department: string;
	    job_title: string;
	    employment_state: string;
	    access_status: string;
	    license_key: string;
	    license_role: string;
	    license_active: boolean;
	    license_assigned_to: string;
	    device_id: string;
	    device_name: string;
	    device_status: string;
	    last_seen_at: string;
	    user_id: string;
	    user_name: string;
	    ready_for_pilot: boolean;
	    issues: string[];
	
	    static createFrom(source: any = {}) {
	        return new PilotReadinessRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.employee_id = source["employee_id"];
	        this.employee_code = source["employee_code"];
	        this.employee_name = source["employee_name"];
	        this.department = source["department"];
	        this.job_title = source["job_title"];
	        this.employment_state = source["employment_state"];
	        this.access_status = source["access_status"];
	        this.license_key = source["license_key"];
	        this.license_role = source["license_role"];
	        this.license_active = source["license_active"];
	        this.license_assigned_to = source["license_assigned_to"];
	        this.device_id = source["device_id"];
	        this.device_name = source["device_name"];
	        this.device_status = source["device_status"];
	        this.last_seen_at = source["last_seen_at"];
	        this.user_id = source["user_id"];
	        this.user_name = source["user_name"];
	        this.ready_for_pilot = source["ready_for_pilot"];
	        this.issues = source["issues"];
	    }
	}
	export class PilotReadinessSummary {
	    generated_at: string;
	    total_employees: number;
	    ready_employees: number;
	    employees_with_issues: number;
	    employees_missing_access: number;
	    activated_licenses: number;
	    unlinked_licenses: number;
	    pending_devices: number;
	    blocked_devices: number;
	    approved_devices: number;
	
	    static createFrom(source: any = {}) {
	        return new PilotReadinessSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.generated_at = source["generated_at"];
	        this.total_employees = source["total_employees"];
	        this.ready_employees = source["ready_employees"];
	        this.employees_with_issues = source["employees_with_issues"];
	        this.employees_missing_access = source["employees_missing_access"];
	        this.activated_licenses = source["activated_licenses"];
	        this.unlinked_licenses = source["unlinked_licenses"];
	        this.pending_devices = source["pending_devices"];
	        this.blocked_devices = source["blocked_devices"];
	        this.approved_devices = source["approved_devices"];
	    }
	}
	export class PilotSupportBundleResult {
	    path: string;
	    rows: number;
	
	    static createFrom(source: any = {}) {
	        return new PilotSupportBundleResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.rows = source["rows"];
	    }
	}
	export class PipelineStage {
	    stage: string;
	    count: number;
	    value: number;
	
	    static createFrom(source: any = {}) {
	        return new PipelineStage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stage = source["stage"];
	        this.count = source["count"];
	        this.value = source["value"];
	    }
	}
	export class PipelineTraceability {
	    order_id: string;
	    order_number: string;
	    rfq_id: string;
	    rfq_found: boolean;
	    rfq_status: string;
	    rfq_customer: string;
	    offer_id: string;
	    offer_number: string;
	    offer_found: boolean;
	    offer_stage: string;
	    offer_value: number;
	
	    static createFrom(source: any = {}) {
	        return new PipelineTraceability(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.order_id = source["order_id"];
	        this.order_number = source["order_number"];
	        this.rfq_id = source["rfq_id"];
	        this.rfq_found = source["rfq_found"];
	        this.rfq_status = source["rfq_status"];
	        this.rfq_customer = source["rfq_customer"];
	        this.offer_id = source["offer_id"];
	        this.offer_number = source["offer_number"];
	        this.offer_found = source["offer_found"];
	        this.offer_stage = source["offer_stage"];
	        this.offer_value = source["offer_value"];
	    }
	}
	export class PricingRecommendation {
	    customer: string;
	    recommended_margin: number;
	    strategy: string;
	    reasoning: string;
	    risk_level: string;
	    confidence_score: number;
	    alternative_margins: AlternativeMargin[];
	    generated_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new PricingRecommendation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.customer = source["customer"];
	        this.recommended_margin = source["recommended_margin"];
	        this.strategy = source["strategy"];
	        this.reasoning = source["reasoning"];
	        this.risk_level = source["risk_level"];
	        this.confidence_score = source["confidence_score"];
	        this.alternative_margins = this.convertValues(source["alternative_margins"], AlternativeMargin);
	        this.generated_at = this.convertValues(source["generated_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ProfitLossReport {
	    period_start: string;
	    period_end: string;
	    revenue: AccountBalance[];
	    total_revenue: number;
	    cogs: AccountBalance[];
	    total_cogs: number;
	    gross_profit: number;
	    gross_margin: number;
	    expenses: AccountBalance[];
	    total_expenses: number;
	    net_income: number;
	    net_margin: number;
	
	    static createFrom(source: any = {}) {
	        return new ProfitLossReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.period_start = source["period_start"];
	        this.period_end = source["period_end"];
	        this.revenue = this.convertValues(source["revenue"], AccountBalance);
	        this.total_revenue = source["total_revenue"];
	        this.cogs = this.convertValues(source["cogs"], AccountBalance);
	        this.total_cogs = source["total_cogs"];
	        this.gross_profit = source["gross_profit"];
	        this.gross_margin = source["gross_margin"];
	        this.expenses = this.convertValues(source["expenses"], AccountBalance);
	        this.total_expenses = source["total_expenses"];
	        this.net_income = source["net_income"];
	        this.net_margin = source["net_margin"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProformaInvoiceItemInput {
	    description: string;
	    quantity: number;
	    rate: number;
	
	    static createFrom(source: any = {}) {
	        return new ProformaInvoiceItemInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.description = source["description"];
	        this.quantity = source["quantity"];
	        this.rate = source["rate"];
	    }
	}
	export class Project {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    name: string;
	    project_type: string;
	    description: string;
	    status: string;
	    customer_id?: string;
	    opportunity_id?: string;
	    order_id?: string;
	    customer_name: string;
	    end_user_name: string;
	    opportunity_key: string;
	    customer_poc_name: string;
	    customer_poc_email: string;
	    customer_poc_phone: string;
	    starts_on?: time.Time;
	    ends_on?: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.name = source["name"];
	        this.project_type = source["project_type"];
	        this.description = source["description"];
	        this.status = source["status"];
	        this.customer_id = source["customer_id"];
	        this.opportunity_id = source["opportunity_id"];
	        this.order_id = source["order_id"];
	        this.customer_name = source["customer_name"];
	        this.end_user_name = source["end_user_name"];
	        this.opportunity_key = source["opportunity_key"];
	        this.customer_poc_name = source["customer_poc_name"];
	        this.customer_poc_email = source["customer_poc_email"];
	        this.customer_poc_phone = source["customer_poc_phone"];
	        this.starts_on = this.convertValues(source["starts_on"], time.Time);
	        this.ends_on = this.convertValues(source["ends_on"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProjectMember {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    project_id: string;
	    employee_id: string;
	    role: string;
	    allocation_percent: number;
	    is_active: boolean;
	    joined_at?: time.Time;
	    left_at?: time.Time;
	    employee_name?: string;
	    project_name?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProjectMember(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.project_id = source["project_id"];
	        this.employee_id = source["employee_id"];
	        this.role = source["role"];
	        this.allocation_percent = source["allocation_percent"];
	        this.is_active = source["is_active"];
	        this.joined_at = this.convertValues(source["joined_at"], time.Time);
	        this.left_at = this.convertValues(source["left_at"], time.Time);
	        this.employee_name = source["employee_name"];
	        this.project_name = source["project_name"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class QuickCapture {
	    id: number;
	    title: string;
	    content: string;
	    tags: string;
	    priority: string;
	    status: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new QuickCapture(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.content = source["content"];
	        this.tags = source["tags"];
	        this.priority = source["priority"];
	        this.status = source["status"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RFQComment {
	    id: number;
	    rfq_id: number;
	    comment: string;
	    created_by: string;
	    created_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new RFQComment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.rfq_id = source["rfq_id"];
	        this.comment = source["comment"];
	        this.created_by = source["created_by"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RFQData {
	    id: number;
	    rfq_number: string;
	    rfq_ref: string;
	    client: string;
	    project: string;
	    value: number;
	    notes: string;
	    status: string;
	    stage: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    document_hash: string;
	    visit_locations: string;
	    product_details: string;
	    source_doc_path: string;
	
	    static createFrom(source: any = {}) {
	        return new RFQData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.rfq_number = source["rfq_number"];
	        this.rfq_ref = source["rfq_ref"];
	        this.client = source["client"];
	        this.project = source["project"];
	        this.value = source["value"];
	        this.notes = source["notes"];
	        this.status = source["status"];
	        this.stage = source["stage"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.document_hash = source["document_hash"];
	        this.visit_locations = source["visit_locations"];
	        this.product_details = source["product_details"];
	        this.source_doc_path = source["source_doc_path"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RFQUpdateRequest {
	    client: string;
	    project: string;
	    rfq_ref: string;
	    value: number;
	    notes: string;
	    status: string;
	    visit_locations: string;
	    product_details: string;
	    document_hash: string;
	    source_doc_path: string;
	
	    static createFrom(source: any = {}) {
	        return new RFQUpdateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.client = source["client"];
	        this.project = source["project"];
	        this.rfq_ref = source["rfq_ref"];
	        this.value = source["value"];
	        this.notes = source["notes"];
	        this.status = source["status"];
	        this.visit_locations = source["visit_locations"];
	        this.product_details = source["product_details"];
	        this.document_hash = source["document_hash"];
	        this.source_doc_path = source["source_doc_path"];
	    }
	}
	
	export class ReconciliationResult {
	    total_offers: number;
	    matched_offers: number;
	    new_offers: number;
	    won_offers: number;
	    pending_offers: number;
	    total_invoices: number;
	    matched_invoices: number;
	    missing_invoices: number;
	    invoices_with_items: number;
	    customer_matches: Record<string, string>;
	    unmatched_customers: string[];
	    fuzzy_matches: FuzzyMatchResult[];
	    total_pos: number;
	    matched_customer_pos: number;
	    matched_supplier_pos: number;
	    discrepancies: DataDiscrepancy[];
	    processing_time: number;
	    confidence_score: number;
	    ready_to_apply: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ReconciliationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_offers = source["total_offers"];
	        this.matched_offers = source["matched_offers"];
	        this.new_offers = source["new_offers"];
	        this.won_offers = source["won_offers"];
	        this.pending_offers = source["pending_offers"];
	        this.total_invoices = source["total_invoices"];
	        this.matched_invoices = source["matched_invoices"];
	        this.missing_invoices = source["missing_invoices"];
	        this.invoices_with_items = source["invoices_with_items"];
	        this.customer_matches = source["customer_matches"];
	        this.unmatched_customers = source["unmatched_customers"];
	        this.fuzzy_matches = this.convertValues(source["fuzzy_matches"], FuzzyMatchResult);
	        this.total_pos = source["total_pos"];
	        this.matched_customer_pos = source["matched_customer_pos"];
	        this.matched_supplier_pos = source["matched_supplier_pos"];
	        this.discrepancies = this.convertValues(source["discrepancies"], DataDiscrepancy);
	        this.processing_time = source["processing_time"];
	        this.confidence_score = source["confidence_score"];
	        this.ready_to_apply = source["ready_to_apply"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ReconciliationStatusSummary {
	    bank_account_id: string;
	    bank_name: string;
	    account_number: string;
	    currency: string;
	    last_reconciled?: time.Time;
	    last_difference: number;
	    is_reconciled: boolean;
	    days_since_reconciled: number;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new ReconciliationStatusSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bank_account_id = source["bank_account_id"];
	        this.bank_name = source["bank_name"];
	        this.account_number = source["account_number"];
	        this.currency = source["currency"];
	        this.last_reconciled = this.convertValues(source["last_reconciled"], time.Time);
	        this.last_difference = source["last_difference"];
	        this.is_reconciled = source["is_reconciled"];
	        this.days_since_reconciled = source["days_since_reconciled"];
	        this.status = source["status"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ReorderSuggestion {
	    product_id: string;
	    product_code: string;
	    product_name: string;
	    quantity_on_hand: number;
	    reorder_point: number;
	    reorder_qty: number;
	    supplier_id: string;
	    supplier_name: string;
	    estimated_cost_bhd: number;
	    lead_time_days: number;
	    urgency_level: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new ReorderSuggestion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.product_id = source["product_id"];
	        this.product_code = source["product_code"];
	        this.product_name = source["product_name"];
	        this.quantity_on_hand = source["quantity_on_hand"];
	        this.reorder_point = source["reorder_point"];
	        this.reorder_qty = source["reorder_qty"];
	        this.supplier_id = source["supplier_id"];
	        this.supplier_name = source["supplier_name"];
	        this.estimated_cost_bhd = source["estimated_cost_bhd"];
	        this.lead_time_days = source["lead_time_days"];
	        this.urgency_level = source["urgency_level"];
	        this.reason = source["reason"];
	    }
	}
	export class StockMovementSummary {
	    type: string;
	    count: number;
	    value: number;
	
	    static createFrom(source: any = {}) {
	        return new StockMovementSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.count = source["count"];
	        this.value = source["value"];
	    }
	}
	export class TopProduct {
	    name: string;
	    revenue: number;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new TopProduct(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.revenue = source["revenue"];
	        this.count = source["count"];
	    }
	}
	export class ReportData {
	    pipeline?: PipelineStage[];
	    conversion_rate?: number;
	    avg_deal_size?: number;
	    win_rate?: number;
	    top_products?: TopProduct[];
	    grade_distribution?: GradeDistItem[];
	    type_distribution?: CustomerTypeItem[];
	    avg_payment_days?: number;
	    collection_efficiency?: number;
	    orders_by_stage?: OrderStage[];
	    avg_lead_time?: number;
	    on_time_delivery?: number;
	    pending_shipments?: number;
	    total_items?: number;
	    total_value?: number;
	    low_stock_alerts?: number;
	    movements?: StockMovementSummary[];
	    receivables_outstanding?: number;
	    payables_outstanding?: number;
	    avg_monthly_revenue?: number;
	    collection_target?: number;
	    collected?: number;
	    overdue?: OverdueBucket[];
	
	    static createFrom(source: any = {}) {
	        return new ReportData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pipeline = this.convertValues(source["pipeline"], PipelineStage);
	        this.conversion_rate = source["conversion_rate"];
	        this.avg_deal_size = source["avg_deal_size"];
	        this.win_rate = source["win_rate"];
	        this.top_products = this.convertValues(source["top_products"], TopProduct);
	        this.grade_distribution = this.convertValues(source["grade_distribution"], GradeDistItem);
	        this.type_distribution = this.convertValues(source["type_distribution"], CustomerTypeItem);
	        this.avg_payment_days = source["avg_payment_days"];
	        this.collection_efficiency = source["collection_efficiency"];
	        this.orders_by_stage = this.convertValues(source["orders_by_stage"], OrderStage);
	        this.avg_lead_time = source["avg_lead_time"];
	        this.on_time_delivery = source["on_time_delivery"];
	        this.pending_shipments = source["pending_shipments"];
	        this.total_items = source["total_items"];
	        this.total_value = source["total_value"];
	        this.low_stock_alerts = source["low_stock_alerts"];
	        this.movements = this.convertValues(source["movements"], StockMovementSummary);
	        this.receivables_outstanding = source["receivables_outstanding"];
	        this.payables_outstanding = source["payables_outstanding"];
	        this.avg_monthly_revenue = source["avg_monthly_revenue"];
	        this.collection_target = source["collection_target"];
	        this.collected = source["collected"];
	        this.overdue = this.convertValues(source["overdue"], OverdueBucket);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ReportMetadata {
	    name: string;
	    path: string;
	    size: number;
	    created_at: time.Time;
	    mime_type: string;
	
	    static createFrom(source: any = {}) {
	        return new ReportMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.size = source["size"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.mime_type = source["mime_type"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SalesPipelineData {
	    stage: string;
	    count: number;
	    value: number;
	    color: string;
	
	    static createFrom(source: any = {}) {
	        return new SalesPipelineData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stage = source["stage"];
	        this.count = source["count"];
	        this.value = source["value"];
	        this.color = source["color"];
	    }
	}
	export class ScanProgress {
	    phase: string;
	    current_file: string;
	    files_processed: number;
	    total_files: number;
	    percentage: number;
	    messages: string[];
	    last_update: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new ScanProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.phase = source["phase"];
	        this.current_file = source["current_file"];
	        this.files_processed = source["files_processed"];
	        this.total_files = source["total_files"];
	        this.percentage = source["percentage"];
	        this.messages = source["messages"];
	        this.last_update = this.convertValues(source["last_update"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ScanResult {
	    workspace_index_path: string;
	    evidence_extracts_path: string;
	    archaeology_report_path: string;
	    summary: ArchaeologyScanSummary;
	
	    static createFrom(source: any = {}) {
	        return new ScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace_index_path = source["workspace_index_path"];
	        this.evidence_extracts_path = source["evidence_extracts_path"];
	        this.archaeology_report_path = source["archaeology_report_path"];
	        this.summary = this.convertValues(source["summary"], ArchaeologyScanSummary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Statistics {
	    total_predictions: number;
	    grade_distribution: Record<string, number>;
	    avg_confidence: number;
	    avg_predicted_days: number;
	
	    static createFrom(source: any = {}) {
	        return new Statistics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_predictions = source["total_predictions"];
	        this.grade_distribution = source["grade_distribution"];
	        this.avg_confidence = source["avg_confidence"];
	        this.avg_predicted_days = source["avg_predicted_days"];
	    }
	}
	
	export class SupplierInvoiceSummary {
	    id: string;
	    invoice_number: string;
	    invoice_date: time.Time;
	    total_bhd: number;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new SupplierInvoiceSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.invoice_number = source["invoice_number"];
	        this.invoice_date = this.convertValues(source["invoice_date"], time.Time);
	        this.total_bhd = source["total_bhd"];
	        this.status = source["status"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SupplierFullProfile {
	    id: string;
	    supplier_code: string;
	    supplier_name: string;
	    supplier_type: string;
	    tax_id: string;
	    country: string;
	    address: string;
	    primary_contact: string;
	    email: string;
	    phone: string;
	    brands_handled: string[];
	    product_types: string[];
	    bank_name: string;
	    account_number: string;
	    iban: string;
	    swift_code: string;
	    rating: number;
	    lead_time_days: number;
	    on_time_delivery_pct: number;
	    total_purchases: number;
	    total_pos: number;
	    avg_po_value: number;
	    outstanding_bhd: number;
	    overdue_bhd: number;
	    recent_pos: POSummary[];
	    recent_invoices: SupplierInvoiceSummary[];
	    issues: crm.SupplierIssue[];
	    open_issues: number;
	    issue_cost: number;
	    notes: crm.EntityNote[];
	
	    static createFrom(source: any = {}) {
	        return new SupplierFullProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.supplier_code = source["supplier_code"];
	        this.supplier_name = source["supplier_name"];
	        this.supplier_type = source["supplier_type"];
	        this.tax_id = source["tax_id"];
	        this.country = source["country"];
	        this.address = source["address"];
	        this.primary_contact = source["primary_contact"];
	        this.email = source["email"];
	        this.phone = source["phone"];
	        this.brands_handled = source["brands_handled"];
	        this.product_types = source["product_types"];
	        this.bank_name = source["bank_name"];
	        this.account_number = source["account_number"];
	        this.iban = source["iban"];
	        this.swift_code = source["swift_code"];
	        this.rating = source["rating"];
	        this.lead_time_days = source["lead_time_days"];
	        this.on_time_delivery_pct = source["on_time_delivery_pct"];
	        this.total_purchases = source["total_purchases"];
	        this.total_pos = source["total_pos"];
	        this.avg_po_value = source["avg_po_value"];
	        this.outstanding_bhd = source["outstanding_bhd"];
	        this.overdue_bhd = source["overdue_bhd"];
	        this.recent_pos = this.convertValues(source["recent_pos"], POSummary);
	        this.recent_invoices = this.convertValues(source["recent_invoices"], SupplierInvoiceSummary);
	        this.issues = this.convertValues(source["issues"], crm.SupplierIssue);
	        this.open_issues = source["open_issues"];
	        this.issue_cost = source["issue_cost"];
	        this.notes = this.convertValues(source["notes"], crm.EntityNote);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SupplierLeadTimeMetrics {
	    supplier_id: string;
	    supplier_name: string;
	    quoted_lead_time_days: number;
	    actual_lead_time_days: number;
	    lead_time_variance: number;
	    variance_percent: number;
	    total_pos: number;
	    on_time_pos: number;
	    late_pos: number;
	    on_time_rate: number;
	    performance_grade: string;
	    average_lateness_days: number;
	
	    static createFrom(source: any = {}) {
	        return new SupplierLeadTimeMetrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supplier_id = source["supplier_id"];
	        this.supplier_name = source["supplier_name"];
	        this.quoted_lead_time_days = source["quoted_lead_time_days"];
	        this.actual_lead_time_days = source["actual_lead_time_days"];
	        this.lead_time_variance = source["lead_time_variance"];
	        this.variance_percent = source["variance_percent"];
	        this.total_pos = source["total_pos"];
	        this.on_time_pos = source["on_time_pos"];
	        this.late_pos = source["late_pos"];
	        this.on_time_rate = source["on_time_rate"];
	        this.performance_grade = source["performance_grade"];
	        this.average_lateness_days = source["average_lateness_days"];
	    }
	}
	
	export class SurvivalMetrics {
	    cash_runway_days: number;
	    monthly_burn_rate: number;
	    cash_on_hand: number;
	    receivables_total: number;
	    payables_total: number;
	    critical_alerts: number;
	    collection_efficiency: number;
	    last_updated: time.Time;
	    runway_status: string;
	    days_of_runway: number;
	    cash_balance: number;
	    monthly_burn: number;
	    week_collections_actual: number;
	    week_collections_target: number;
	    overdue_by_grade: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new SurvivalMetrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cash_runway_days = source["cash_runway_days"];
	        this.monthly_burn_rate = source["monthly_burn_rate"];
	        this.cash_on_hand = source["cash_on_hand"];
	        this.receivables_total = source["receivables_total"];
	        this.payables_total = source["payables_total"];
	        this.critical_alerts = source["critical_alerts"];
	        this.collection_efficiency = source["collection_efficiency"];
	        this.last_updated = this.convertValues(source["last_updated"], time.Time);
	        this.runway_status = source["runway_status"];
	        this.days_of_runway = source["days_of_runway"];
	        this.cash_balance = source["cash_balance"];
	        this.monthly_burn = source["monthly_burn"];
	        this.week_collections_actual = source["week_collections_actual"];
	        this.week_collections_target = source["week_collections_target"];
	        this.overdue_by_grade = source["overdue_by_grade"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SyncHealth {
	    is_online: boolean;
	    last_sync_at: string;
	    last_sync_status: string;
	    tables_in_sync: number;
	    db_size_bytes: number;
	    backup_count: number;
	    last_backup_at: string;
	    uptime_seconds: number;
	
	    static createFrom(source: any = {}) {
	        return new SyncHealth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.is_online = source["is_online"];
	        this.last_sync_at = source["last_sync_at"];
	        this.last_sync_status = source["last_sync_status"];
	        this.tables_in_sync = source["tables_in_sync"];
	        this.db_size_bytes = source["db_size_bytes"];
	        this.backup_count = source["backup_count"];
	        this.last_backup_at = source["last_backup_at"];
	        this.uptime_seconds = source["uptime_seconds"];
	    }
	}
	export class SystemInfo {
	    os: string;
	    cpu: string;
	    gpu: string;
	    ram: string;
	    has_gpu: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SystemInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.os = source["os"];
	        this.cpu = source["cpu"];
	        this.gpu = source["gpu"];
	        this.ram = source["ram"];
	        this.has_gpu = source["has_gpu"];
	    }
	}
	export class TallyBalanceSheet {
	    as_of_date: time.Time;
	    year: number;
	    generated_at: time.Time;
	    cash: number;
	    accounts_receivable: number;
	    inventory: number;
	    total_current_assets: number;
	    total_assets: number;
	    accounts_payable: number;
	    total_current_liabilities: number;
	    total_liabilities: number;
	    retained_earnings: number;
	    total_equity: number;
	    currency: string;
	
	    static createFrom(source: any = {}) {
	        return new TallyBalanceSheet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.as_of_date = this.convertValues(source["as_of_date"], time.Time);
	        this.year = source["year"];
	        this.generated_at = this.convertValues(source["generated_at"], time.Time);
	        this.cash = source["cash"];
	        this.accounts_receivable = source["accounts_receivable"];
	        this.inventory = source["inventory"];
	        this.total_current_assets = source["total_current_assets"];
	        this.total_assets = source["total_assets"];
	        this.accounts_payable = source["accounts_payable"];
	        this.total_current_liabilities = source["total_current_liabilities"];
	        this.total_liabilities = source["total_liabilities"];
	        this.retained_earnings = source["retained_earnings"];
	        this.total_equity = source["total_equity"];
	        this.currency = source["currency"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TallyImportResult {
	    total_rows: number;
	    imported: number;
	    duplicates: number;
	    errors: number;
	    error_details?: string[];
	    batch_id: string;
	
	    static createFrom(source: any = {}) {
	        return new TallyImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_rows = source["total_rows"];
	        this.imported = source["imported"];
	        this.duplicates = source["duplicates"];
	        this.errors = source["errors"];
	        this.error_details = source["error_details"];
	        this.batch_id = source["batch_id"];
	    }
	}
	export class TallyPLReport {
	    year: number;
	    generated_at: time.Time;
	    sales_revenue: number;
	    other_income: number;
	    total_revenue: number;
	    purchases: number;
	    cost_of_goods_sold: number;
	    gross_profit: number;
	    gross_profit_margin: number;
	    operating_expenses: number;
	    net_profit: number;
	    net_profit_margin: number;
	    invoice_count: number;
	    purchase_count: number;
	    currency: string;
	    monthly_breakdown?: MonthlyPLBreakdown[];
	
	    static createFrom(source: any = {}) {
	        return new TallyPLReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year = source["year"];
	        this.generated_at = this.convertValues(source["generated_at"], time.Time);
	        this.sales_revenue = source["sales_revenue"];
	        this.other_income = source["other_income"];
	        this.total_revenue = source["total_revenue"];
	        this.purchases = source["purchases"];
	        this.cost_of_goods_sold = source["cost_of_goods_sold"];
	        this.gross_profit = source["gross_profit"];
	        this.gross_profit_margin = source["gross_profit_margin"];
	        this.operating_expenses = source["operating_expenses"];
	        this.net_profit = source["net_profit"];
	        this.net_profit_margin = source["net_profit_margin"];
	        this.invoice_count = source["invoice_count"];
	        this.purchase_count = source["purchase_count"];
	        this.currency = source["currency"];
	        this.monthly_breakdown = this.convertValues(source["monthly_breakdown"], MonthlyPLBreakdown);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TaskActivity {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    task_id: string;
	    employee_id: string;
	    activity_type: string;
	    detail: string;
	    metadata_json: string;
	    employee_name?: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskActivity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.task_id = source["task_id"];
	        this.employee_id = source["employee_id"];
	        this.activity_type = source["activity_type"];
	        this.detail = source["detail"];
	        this.metadata_json = source["metadata_json"];
	        this.employee_name = source["employee_name"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TaskComment {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    task_id: string;
	    employee_id: string;
	    body: string;
	    employee_name?: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskComment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.task_id = source["task_id"];
	        this.employee_id = source["employee_id"];
	        this.body = source["body"];
	        this.employee_name = source["employee_name"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TaskItem {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    title: string;
	    description: string;
	    task_type: string;
	    legacy_follow_up_id?: string;
	    status: string;
	    blocked_reason: string;
	    priority: string;
	    due_date?: time.Time;
	    customer_id?: string;
	    opportunity_id?: string;
	    order_id?: string;
	    project_id?: string;
	    creator_employee_id: string;
	    assignee_employee_id?: string;
	    watchers_json: string;
	    started_at?: time.Time;
	    completed_at?: time.Time;
	    last_comment_at?: time.Time;
	    creator_name?: string;
	    assignee_name?: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.task_type = source["task_type"];
	        this.legacy_follow_up_id = source["legacy_follow_up_id"];
	        this.status = source["status"];
	        this.blocked_reason = source["blocked_reason"];
	        this.priority = source["priority"];
	        this.due_date = this.convertValues(source["due_date"], time.Time);
	        this.customer_id = source["customer_id"];
	        this.opportunity_id = source["opportunity_id"];
	        this.order_id = source["order_id"];
	        this.project_id = source["project_id"];
	        this.creator_employee_id = source["creator_employee_id"];
	        this.assignee_employee_id = source["assignee_employee_id"];
	        this.watchers_json = source["watchers_json"];
	        this.started_at = this.convertValues(source["started_at"], time.Time);
	        this.completed_at = this.convertValues(source["completed_at"], time.Time);
	        this.last_comment_at = this.convertValues(source["last_comment_at"], time.Time);
	        this.creator_name = source["creator_name"];
	        this.assignee_name = source["assignee_name"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TenderFolderPreview {
	    folder_path: string;
	    folder_name: string;
	    workflow_key: string;
	    title: string;
	    existing: boolean;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new TenderFolderPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folder_path = source["folder_path"];
	        this.folder_name = source["folder_name"];
	        this.workflow_key = source["workflow_key"];
	        this.title = source["title"];
	        this.existing = source["existing"];
	        this.status = source["status"];
	    }
	}
	export class ThreeWayMatchResult {
	    matched: boolean;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new ThreeWayMatchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.matched = source["matched"];
	        this.reason = source["reason"];
	    }
	}
	
	export class UnifiedThreadEntry {
	    id: string;
	    source_type: string;
	    source_id: string;
	    workflow_key: string;
	    comment: string;
	    created_by: string;
	    created_at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new UnifiedThreadEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source_type = source["source_type"];
	        this.source_id = source["source_id"];
	        this.workflow_key = source["workflow_key"];
	        this.comment = source["comment"];
	        this.created_by = source["created_by"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UserActivityChartRow {
	    label: string;
	    active_hours: number;
	    meaningful_hours: number;
	    efficiency_score: number;
	
	    static createFrom(source: any = {}) {
	        return new UserActivityChartRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.active_hours = source["active_hours"];
	        this.meaningful_hours = source["meaningful_hours"];
	        this.efficiency_score = source["efficiency_score"];
	    }
	}
	export class UserActivityEventInput {
	    session_id: string;
	    event_time: string;
	    event_type: string;
	    category: string;
	    screen: string;
	    route: string;
	    action_label: string;
	    action_key: string;
	    resource_type: string;
	    resource_id: string;
	    search_text: string;
	    metadata: Record<string, any>;
	    active_seconds: number;
	    meaningful_seconds: number;
	    idle_seconds: number;
	
	    static createFrom(source: any = {}) {
	        return new UserActivityEventInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.session_id = source["session_id"];
	        this.event_time = source["event_time"];
	        this.event_type = source["event_type"];
	        this.category = source["category"];
	        this.screen = source["screen"];
	        this.route = source["route"];
	        this.action_label = source["action_label"];
	        this.action_key = source["action_key"];
	        this.resource_type = source["resource_type"];
	        this.resource_id = source["resource_id"];
	        this.search_text = source["search_text"];
	        this.metadata = source["metadata"];
	        this.active_seconds = source["active_seconds"];
	        this.meaningful_seconds = source["meaningful_seconds"];
	        this.idle_seconds = source["idle_seconds"];
	    }
	}
	export class UserActivityHeartbeatInput {
	    session_id: string;
	    screen: string;
	    active_seconds: number;
	    meaningful_seconds: number;
	    idle_seconds: number;
	    event_count: number;
	    search_count: number;
	    create_count: number;
	    update_count: number;
	    export_count: number;
	    navigation_count: number;
	
	    static createFrom(source: any = {}) {
	        return new UserActivityHeartbeatInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.session_id = source["session_id"];
	        this.screen = source["screen"];
	        this.active_seconds = source["active_seconds"];
	        this.meaningful_seconds = source["meaningful_seconds"];
	        this.idle_seconds = source["idle_seconds"];
	        this.event_count = source["event_count"];
	        this.search_count = source["search_count"];
	        this.create_count = source["create_count"];
	        this.update_count = source["update_count"];
	        this.export_count = source["export_count"];
	        this.navigation_count = source["navigation_count"];
	    }
	}
	export class UserActivityMetric {
	    key: string;
	    label: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new UserActivityMetric(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.label = source["label"];
	        this.count = source["count"];
	    }
	}
	export class UserActivitySession {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    session_id: string;
	    started_at: time.Time;
	    ended_at?: time.Time;
	    last_seen_at: time.Time;
	    user_id: string;
	    employee_id: string;
	    employee_name: string;
	    license_key_hash: string;
	    license_role: string;
	    device_hash: string;
	    source: string;
	    primary_screen: string;
	    is_open: boolean;
	    active_seconds: number;
	    meaningful_seconds: number;
	    idle_seconds: number;
	    event_count: number;
	    search_count: number;
	    create_count: number;
	    update_count: number;
	    export_count: number;
	    navigation_count: number;
	
	    static createFrom(source: any = {}) {
	        return new UserActivitySession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.session_id = source["session_id"];
	        this.started_at = this.convertValues(source["started_at"], time.Time);
	        this.ended_at = this.convertValues(source["ended_at"], time.Time);
	        this.last_seen_at = this.convertValues(source["last_seen_at"], time.Time);
	        this.user_id = source["user_id"];
	        this.employee_id = source["employee_id"];
	        this.employee_name = source["employee_name"];
	        this.license_key_hash = source["license_key_hash"];
	        this.license_role = source["license_role"];
	        this.device_hash = source["device_hash"];
	        this.source = source["source"];
	        this.primary_screen = source["primary_screen"];
	        this.is_open = source["is_open"];
	        this.active_seconds = source["active_seconds"];
	        this.meaningful_seconds = source["meaningful_seconds"];
	        this.idle_seconds = source["idle_seconds"];
	        this.event_count = source["event_count"];
	        this.search_count = source["search_count"];
	        this.create_count = source["create_count"];
	        this.update_count = source["update_count"];
	        this.export_count = source["export_count"];
	        this.navigation_count = source["navigation_count"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UserActivityUserReport {
	    user_key: string;
	    user_id: string;
	    employee_id: string;
	    employee_name: string;
	    license_role: string;
	    active_hours: number;
	    meaningful_hours: number;
	    idle_hours: number;
	    efficiency_score: number;
	    event_count: number;
	    search_count: number;
	    create_count: number;
	    update_count: number;
	    export_count: number;
	    navigation_count: number;
	    top_screens: UserActivityMetric[];
	    top_actions: UserActivityMetric[];
	    top_searches: UserActivityMetric[];
	    last_activity_at: string;
	
	    static createFrom(source: any = {}) {
	        return new UserActivityUserReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user_key = source["user_key"];
	        this.user_id = source["user_id"];
	        this.employee_id = source["employee_id"];
	        this.employee_name = source["employee_name"];
	        this.license_role = source["license_role"];
	        this.active_hours = source["active_hours"];
	        this.meaningful_hours = source["meaningful_hours"];
	        this.idle_hours = source["idle_hours"];
	        this.efficiency_score = source["efficiency_score"];
	        this.event_count = source["event_count"];
	        this.search_count = source["search_count"];
	        this.create_count = source["create_count"];
	        this.update_count = source["update_count"];
	        this.export_count = source["export_count"];
	        this.navigation_count = source["navigation_count"];
	        this.top_screens = this.convertValues(source["top_screens"], UserActivityMetric);
	        this.top_actions = this.convertValues(source["top_actions"], UserActivityMetric);
	        this.top_searches = this.convertValues(source["top_searches"], UserActivityMetric);
	        this.last_activity_at = source["last_activity_at"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UserActivityWeeklyReport {
	    week_start: string;
	    week_end: string;
	    generated_at: string;
	    total_active_hours: number;
	    total_meaningful_hours: number;
	    average_efficiency: number;
	    user_count: number;
	    users: UserActivityUserReport[];
	    chart_rows: UserActivityChartRow[];
	    monitoring_principals: string[];
	    confidentiality_notice: string;
	
	    static createFrom(source: any = {}) {
	        return new UserActivityWeeklyReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.week_start = source["week_start"];
	        this.week_end = source["week_end"];
	        this.generated_at = source["generated_at"];
	        this.total_active_hours = source["total_active_hours"];
	        this.total_meaningful_hours = source["total_meaningful_hours"];
	        this.average_efficiency = source["average_efficiency"];
	        this.user_count = source["user_count"];
	        this.users = this.convertValues(source["users"], UserActivityUserReport);
	        this.chart_rows = this.convertValues(source["chart_rows"], UserActivityChartRow);
	        this.monitoring_principals = source["monitoring_principals"];
	        this.confidentiality_notice = source["confidentiality_notice"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class VATReconciliation {
	    start_date: time.Time;
	    end_date: time.Time;
	    output_vat: number;
	    input_vat: number;
	    net_vat: number;
	    customer_invoices: number;
	    supplier_invoices: number;
	
	    static createFrom(source: any = {}) {
	        return new VATReconciliation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start_date = this.convertValues(source["start_date"], time.Time);
	        this.end_date = this.convertValues(source["end_date"], time.Time);
	        this.output_vat = source["output_vat"];
	        this.input_vat = source["input_vat"];
	        this.net_vat = source["net_vat"];
	        this.customer_invoices = source["customer_invoices"];
	        this.supplier_invoices = source["supplier_invoices"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ValidationResult {
	    valid: boolean;
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new ValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.errors = source["errors"];
	    }
	}
	export class VarianceItem {
	    category: string;
	    description: string;
	    amount: number;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new VarianceItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.category = source["category"];
	        this.description = source["description"];
	        this.amount = source["amount"];
	        this.type = source["type"];
	    }
	}

}

export namespace payroll {
	
	export class CompensationProfile {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    employee_id: string;
	    division: string;
	    pay_frequency: string;
	    currency: string;
	    base_salary: number;
	    housing_allowance: number;
	    transport_allowance: number;
	    other_allowance: number;
	    standard_deduction: number;
	    tax_deduction: number;
	    employer_cost: number;
	    effective_from?: time.Time;
	    effective_to?: time.Time;
	    is_active: boolean;
	    notes: string;
	    employee_name?: string;
	    job_title?: string;
	
	    static createFrom(source: any = {}) {
	        return new CompensationProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.employee_id = source["employee_id"];
	        this.division = source["division"];
	        this.pay_frequency = source["pay_frequency"];
	        this.currency = source["currency"];
	        this.base_salary = source["base_salary"];
	        this.housing_allowance = source["housing_allowance"];
	        this.transport_allowance = source["transport_allowance"];
	        this.other_allowance = source["other_allowance"];
	        this.standard_deduction = source["standard_deduction"];
	        this.tax_deduction = source["tax_deduction"];
	        this.employer_cost = source["employer_cost"];
	        this.effective_from = this.convertValues(source["effective_from"], time.Time);
	        this.effective_to = this.convertValues(source["effective_to"], time.Time);
	        this.is_active = source["is_active"];
	        this.notes = source["notes"];
	        this.employee_name = source["employee_name"];
	        this.job_title = source["job_title"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Component {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    payroll_run_item_id: string;
	    component_type: string;
	    code: string;
	    name: string;
	    amount: number;
	
	    static createFrom(source: any = {}) {
	        return new Component(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.payroll_run_item_id = source["payroll_run_item_id"];
	        this.component_type = source["component_type"];
	        this.code = source["code"];
	        this.name = source["name"];
	        this.amount = source["amount"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DashboardSummary {
	    active_profiles: number;
	    open_periods: number;
	    draft_runs: number;
	    approved_unpaid_runs: number;
	    month_to_date_net_payroll: number;
	    upcoming_payroll_liability: number;
	
	    static createFrom(source: any = {}) {
	        return new DashboardSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.active_profiles = source["active_profiles"];
	        this.open_periods = source["open_periods"];
	        this.draft_runs = source["draft_runs"];
	        this.approved_unpaid_runs = source["approved_unpaid_runs"];
	        this.month_to_date_net_payroll = source["month_to_date_net_payroll"];
	        this.upcoming_payroll_liability = source["upcoming_payroll_liability"];
	    }
	}
	export class Payout {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    payroll_run_id: string;
	    payroll_run_item_id: string;
	    employee_id: string;
	    division: string;
	    scheduled_at?: time.Time;
	    paid_at?: time.Time;
	    amount: number;
	    currency: string;
	    status: string;
	    payment_reference: string;
	    bank_account_id?: string;
	    bank_statement_line_id?: string;
	    employee_name?: string;
	    run_number?: string;
	
	    static createFrom(source: any = {}) {
	        return new Payout(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.payroll_run_id = source["payroll_run_id"];
	        this.payroll_run_item_id = source["payroll_run_item_id"];
	        this.employee_id = source["employee_id"];
	        this.division = source["division"];
	        this.scheduled_at = this.convertValues(source["scheduled_at"], time.Time);
	        this.paid_at = this.convertValues(source["paid_at"], time.Time);
	        this.amount = source["amount"];
	        this.currency = source["currency"];
	        this.status = source["status"];
	        this.payment_reference = source["payment_reference"];
	        this.bank_account_id = source["bank_account_id"];
	        this.bank_statement_line_id = source["bank_statement_line_id"];
	        this.employee_name = source["employee_name"];
	        this.run_number = source["run_number"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Period {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    name: string;
	    division: string;
	    period_start: time.Time;
	    period_end: time.Time;
	    payment_date?: time.Time;
	    status: string;
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new Period(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.name = source["name"];
	        this.division = source["division"];
	        this.period_start = this.convertValues(source["period_start"], time.Time);
	        this.period_end = this.convertValues(source["period_end"], time.Time);
	        this.payment_date = this.convertValues(source["payment_date"], time.Time);
	        this.status = source["status"];
	        this.notes = source["notes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RunItem {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    payroll_run_id: string;
	    employee_id: string;
	    compensation_profile_id?: string;
	    employee_name_snapshot: string;
	    job_title_snapshot: string;
	    base_salary: number;
	    allowances_total: number;
	    deductions_total: number;
	    employer_cost_total: number;
	    gross_pay: number;
	    net_pay: number;
	    status: string;
	    notes: string;
	    employee_name?: string;
	    payout_id?: string;
	    payout_status?: string;
	    payout_paid_at?: time.Time;
	    components?: Component[];
	
	    static createFrom(source: any = {}) {
	        return new RunItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.payroll_run_id = source["payroll_run_id"];
	        this.employee_id = source["employee_id"];
	        this.compensation_profile_id = source["compensation_profile_id"];
	        this.employee_name_snapshot = source["employee_name_snapshot"];
	        this.job_title_snapshot = source["job_title_snapshot"];
	        this.base_salary = source["base_salary"];
	        this.allowances_total = source["allowances_total"];
	        this.deductions_total = source["deductions_total"];
	        this.employer_cost_total = source["employer_cost_total"];
	        this.gross_pay = source["gross_pay"];
	        this.net_pay = source["net_pay"];
	        this.status = source["status"];
	        this.notes = source["notes"];
	        this.employee_name = source["employee_name"];
	        this.payout_id = source["payout_id"];
	        this.payout_status = source["payout_status"];
	        this.payout_paid_at = this.convertValues(source["payout_paid_at"], time.Time);
	        this.components = this.convertValues(source["components"], Component);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Run {
	    id: string;
	    created_at: time.Time;
	    updated_at: time.Time;
	    version: number;
	    created_by: string;
	    run_number: string;
	    payroll_period_id: string;
	    division: string;
	    status: string;
	    generated_at?: time.Time;
	    approved_at?: time.Time;
	    approved_by: string;
	    posted_at?: time.Time;
	    posted_by: string;
	    paid_at?: time.Time;
	    payment_reference: string;
	    bank_account_id?: string;
	    journal_entry_id?: string;
	    payout_journal_entry_id?: string;
	    total_employees: number;
	    gross_total: number;
	    deductions_total: number;
	    net_total: number;
	    employer_cost_total: number;
	    currency: string;
	    notes: string;
	    period_name?: string;
	    items?: RunItem[];
	    payouts?: Payout[];
	
	    static createFrom(source: any = {}) {
	        return new Run(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.created_at = this.convertValues(source["created_at"], time.Time);
	        this.updated_at = this.convertValues(source["updated_at"], time.Time);
	        this.version = source["version"];
	        this.created_by = source["created_by"];
	        this.run_number = source["run_number"];
	        this.payroll_period_id = source["payroll_period_id"];
	        this.division = source["division"];
	        this.status = source["status"];
	        this.generated_at = this.convertValues(source["generated_at"], time.Time);
	        this.approved_at = this.convertValues(source["approved_at"], time.Time);
	        this.approved_by = source["approved_by"];
	        this.posted_at = this.convertValues(source["posted_at"], time.Time);
	        this.posted_by = source["posted_by"];
	        this.paid_at = this.convertValues(source["paid_at"], time.Time);
	        this.payment_reference = source["payment_reference"];
	        this.bank_account_id = source["bank_account_id"];
	        this.journal_entry_id = source["journal_entry_id"];
	        this.payout_journal_entry_id = source["payout_journal_entry_id"];
	        this.total_employees = source["total_employees"];
	        this.gross_total = source["gross_total"];
	        this.deductions_total = source["deductions_total"];
	        this.net_total = source["net_total"];
	        this.employer_cost_total = source["employer_cost_total"];
	        this.currency = source["currency"];
	        this.notes = source["notes"];
	        this.period_name = source["period_name"];
	        this.items = this.convertValues(source["items"], RunItem);
	        this.payouts = this.convertValues(source["payouts"], Payout);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace posting {
	
	export class AccountRef {
	    role: string;
	    code: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new AccountRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.code = source["code"];
	        this.name = source["name"];
	    }
	}
	export class CoverageRow {
	    source_type: string;
	    label: string;
	    total: number;
	    linked: number;
	    missing: number;
	    draft_entries: number;
	    is_complete: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CoverageRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_type = source["source_type"];
	        this.label = source["label"];
	        this.total = source["total"];
	        this.linked = source["linked"];
	        this.missing = source["missing"];
	        this.draft_entries = source["draft_entries"];
	        this.is_complete = source["is_complete"];
	    }
	}
	export class CoverageReport {
	    rows: CoverageRow[];
	    total: number;
	    linked: number;
	    missing: number;
	    draft_entries: number;
	    is_complete: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CoverageReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rows = this.convertValues(source["rows"], CoverageRow);
	        this.total = source["total"];
	        this.linked = source["linked"];
	        this.missing = source["missing"];
	        this.draft_entries = source["draft_entries"];
	        this.is_complete = source["is_complete"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Line {
	    account: AccountRef;
	    debit: number;
	    credit: number;
	    memo: string;
	
	    static createFrom(source: any = {}) {
	        return new Line(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.account = this.convertValues(source["account"], AccountRef);
	        this.debit = source["debit"];
	        this.credit = source["credit"];
	        this.memo = source["memo"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Entry {
	    source_type: string;
	    source_id: string;
	    source_number: string;
	    entry_date: time.Time;
	    description: string;
	    currency: string;
	    debit_total: number;
	    credit_total: number;
	    is_balanced: boolean;
	    lines: Line[];
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_type = source["source_type"];
	        this.source_id = source["source_id"];
	        this.source_number = source["source_number"];
	        this.entry_date = this.convertValues(source["entry_date"], time.Time);
	        this.description = source["description"];
	        this.currency = source["currency"];
	        this.debit_total = source["debit_total"];
	        this.credit_total = source["credit_total"];
	        this.is_balanced = source["is_balanced"];
	        this.lines = this.convertValues(source["lines"], Line);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class TrialBalanceRow {
	    account: AccountRef;
	    debit: number;
	    credit: number;
	    net: number;
	
	    static createFrom(source: any = {}) {
	        return new TrialBalanceRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.account = this.convertValues(source["account"], AccountRef);
	        this.debit = source["debit"];
	        this.credit = source["credit"];
	        this.net = source["net"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TrialBalanceGate {
	    fiscal_year: number;
	    fiscal_period: number;
	    entry_count: number;
	    line_count: number;
	    debit_total: number;
	    credit_total: number;
	    difference: number;
	    is_balanced: boolean;
	    rows: TrialBalanceRow[];
	    balanced_accounts?: string[];
	    imbalanced_accounts?: string[];
	
	    static createFrom(source: any = {}) {
	        return new TrialBalanceGate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fiscal_year = source["fiscal_year"];
	        this.fiscal_period = source["fiscal_period"];
	        this.entry_count = source["entry_count"];
	        this.line_count = source["line_count"];
	        this.debit_total = source["debit_total"];
	        this.credit_total = source["credit_total"];
	        this.difference = source["difference"];
	        this.is_balanced = source["is_balanced"];
	        this.rows = this.convertValues(source["rows"], TrialBalanceRow);
	        this.balanced_accounts = source["balanced_accounts"];
	        this.imbalanced_accounts = source["imbalanced_accounts"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace prediction {
	
	export class BatchSummary {
	    total_customers: number;
	    grade_distribution: Record<string, number>;
	    avg_payment_days: number;
	    avg_confidence: number;
	    total_value: number;
	    approved_value: number;
	    rejected_value: number;
	    approval_rate: number;
	
	    static createFrom(source: any = {}) {
	        return new BatchSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_customers = source["total_customers"];
	        this.grade_distribution = source["grade_distribution"];
	        this.avg_payment_days = source["avg_payment_days"];
	        this.avg_confidence = source["avg_confidence"];
	        this.total_value = source["total_value"];
	        this.approved_value = source["approved_value"];
	        this.rejected_value = source["rejected_value"];
	        this.approval_rate = source["approval_rate"];
	    }
	}
	export class Customer {
	    id: string;
	    business_name: string;
	    order_value: number;
	    order_history: number[];
	    payment_history: number[];
	    relation_years: number;
	    industry: string;
	    country: string;
	    is_emergency: number;
	    has_abb: number;
	    dispute_count: number;
	
	    static createFrom(source: any = {}) {
	        return new Customer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.business_name = source["business_name"];
	        this.order_value = source["order_value"];
	        this.order_history = source["order_history"];
	        this.payment_history = source["payment_history"];
	        this.relation_years = source["relation_years"];
	        this.industry = source["industry"];
	        this.country = source["country"];
	        this.is_emergency = source["is_emergency"];
	        this.has_abb = source["has_abb"];
	        this.dispute_count = source["dispute_count"];
	    }
	}
	export class ThreeRegime {
	    r1: number;
	    r2: number;
	    r3: number;
	
	    static createFrom(source: any = {}) {
	        return new ThreeRegime(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.r1 = source["r1"];
	        this.r2 = source["r2"];
	        this.r3 = source["r3"];
	    }
	}
	export class PaymentPrediction {
	    customer_id: string;
	    customer_name: string;
	    grade: string;
	    predicted_days: number;
	    confidence: number;
	    three_regimes: ThreeRegime;
	    regimes: ThreeRegime;
	    risk_factors: string[];
	    recommended_action: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new PaymentPrediction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.customer_id = source["customer_id"];
	        this.customer_name = source["customer_name"];
	        this.grade = source["grade"];
	        this.predicted_days = source["predicted_days"];
	        this.confidence = source["confidence"];
	        this.three_regimes = this.convertValues(source["three_regimes"], ThreeRegime);
	        this.regimes = this.convertValues(source["regimes"], ThreeRegime);
	        this.risk_factors = source["risk_factors"];
	        this.recommended_action = source["recommended_action"];
	        this.timestamp = source["timestamp"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace release {
	
	export class BuildInfo {
	    product: string;
	    version: string;
	    channel: string;
	    release_date: string;
	    release_name: string;
	    minimum_supported_version: string;
	    schema_version: string;
	    notes: string;
	    git_commit: string;
	    build_time: string;
	    dirty: boolean;
	    go_version: string;
	    goos: string;
	    goarch: string;
	
	    static createFrom(source: any = {}) {
	        return new BuildInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.product = source["product"];
	        this.version = source["version"];
	        this.channel = source["channel"];
	        this.release_date = source["release_date"];
	        this.release_name = source["release_name"];
	        this.minimum_supported_version = source["minimum_supported_version"];
	        this.schema_version = source["schema_version"];
	        this.notes = source["notes"];
	        this.git_commit = source["git_commit"];
	        this.build_time = source["build_time"];
	        this.dirty = source["dirty"];
	        this.go_version = source["go_version"];
	        this.goos = source["goos"];
	        this.goarch = source["goarch"];
	    }
	}

}

export namespace shared {
	
	export class StatusBadgeVM {
	    label: string;
	    color: string;
	    icon?: string;
	    tooltip?: string;
	
	    static createFrom(source: any = {}) {
	        return new StatusBadgeVM(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.color = source["color"];
	        this.icon = source["icon"];
	        this.tooltip = source["tooltip"];
	    }
	}
	export class TableColumn {
	    key: string;
	    label: string;
	    type: string;
	    sortable: boolean;
	    width?: string;
	    align?: string;
	    currency?: string;
	
	    static createFrom(source: any = {}) {
	        return new TableColumn(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.label = source["label"];
	        this.type = source["type"];
	        this.sortable = source["sortable"];
	        this.width = source["width"];
	        this.align = source["align"];
	        this.currency = source["currency"];
	    }
	}
	export class TableFilter {
	    column: string;
	    type: string;
	    value?: any;
	    options?: viewmodel.Option[];
	
	    static createFrom(source: any = {}) {
	        return new TableFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.column = source["column"];
	        this.type = source["type"];
	        this.value = source["value"];
	        this.options = this.convertValues(source["options"], viewmodel.Option);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TableRow {
	    id: string;
	    fields: Record<string, any>;
	    actions?: viewmodel.ActionButton[];
	    status?: string;
	
	    static createFrom(source: any = {}) {
	        return new TableRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.fields = source["fields"];
	        this.actions = this.convertValues(source["actions"], viewmodel.ActionButton);
	        this.status = source["status"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TableVM {
	    columns: TableColumn[];
	    rows: TableRow[];
	    totalRows: number;
	    page: number;
	    pageSize: number;
	    sortColumn: string;
	    sortDesc: boolean;
	    filters?: TableFilter[];
	
	    static createFrom(source: any = {}) {
	        return new TableVM(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.columns = this.convertValues(source["columns"], TableColumn);
	        this.rows = this.convertValues(source["rows"], TableRow);
	        this.totalRows = source["totalRows"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	        this.sortColumn = source["sortColumn"];
	        this.sortDesc = source["sortDesc"];
	        this.filters = this.convertValues(source["filters"], TableFilter);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace time {
	
	export class Time {
	
	
	    static createFrom(source: any = {}) {
	        return new Time(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

export namespace viewmodel {
	
	export class ActionButton {
	    label: string;
	    action: string;
	    icon?: string;
	    variant?: string;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ActionButton(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.action = source["action"];
	        this.icon = source["icon"];
	        this.variant = source["variant"];
	        this.enabled = source["enabled"];
	    }
	}
	export class ComplianceIssue {
	    severity: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ComplianceIssue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.severity = source["severity"];
	        this.message = source["message"];
	    }
	}
	export class ValidationEntry {
	    timestamp: string;
	    event_name: string;
	    jurisdiction: string;
	    valid: boolean;
	    errors: string[];
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new ValidationEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.event_name = source["event_name"];
	        this.jurisdiction = source["jurisdiction"];
	        this.valid = source["valid"];
	        this.errors = source["errors"];
	        this.warnings = source["warnings"];
	    }
	}
	export class ComplianceDashboardVM {
	    jurisdiction: string;
	    tax_rates: compliance.TaxRate[];
	    recent_validations: ValidationEntry[];
	    compliance_score: number;
	    issues: ComplianceIssue[];
	
	    static createFrom(source: any = {}) {
	        return new ComplianceDashboardVM(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jurisdiction = source["jurisdiction"];
	        this.tax_rates = this.convertValues(source["tax_rates"], compliance.TaxRate);
	        this.recent_validations = this.convertValues(source["recent_validations"], ValidationEntry);
	        this.compliance_score = source["compliance_score"];
	        this.issues = this.convertValues(source["issues"], ComplianceIssue);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Option {
	    value: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new Option(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.label = source["label"];
	    }
	}

}

