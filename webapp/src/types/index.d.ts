export interface LegalHold {
    id: string;
    name: string;
    display_name: string;
}

export interface CreateLegalHold {
    display_name: string;
    starts_at: number;
    ends_at: number;
    users: Array<string>;
}
