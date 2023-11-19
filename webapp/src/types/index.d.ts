export interface LegalHold {
    id: string;
    name: string;
    display_name: string;
    user_ids: string[];
}

export interface CreateLegalHold {
    name: string;
    display_name: string;
    starts_at: number;
    ends_at: number;
    user_ids: Array<string>;
}
