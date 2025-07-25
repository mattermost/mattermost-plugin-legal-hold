export interface LegalHold {
    id: string;
    name: string;
    display_name: string;
    starts_at: number;
    ends_at: number;
    user_ids: string[];
    group_ids: string[];
    include_public_channels: boolean;
    secret: string;
    last_execution_ended_at: number;
    has_messages: boolean;
    status: 'idle' | 'executing';
}

export interface CreateLegalHold {
    name: string;
    display_name: string;
    starts_at: number;
    ends_at: number;
    user_ids: Array<string>;
    group_ids?: Array<string>;
}

export interface UpdateLegalHold {
    id: string;
    display_name: string;
    ends_at: number;
    user_ids: Array<string>;
    group_ids?: Array<string>;
}
