import {useEffect, useMemo} from 'react';
import {useDispatch} from 'react-redux';

import {getMissingProfilesByIds} from 'mattermost-redux/actions/users';

import {getMissingGroupsByIds} from '@/actions/groups';
import LegalHoldRow from '@/components/legal_hold_table/legal_hold_row';
import type {LegalHold} from '@/types';

import './legal_hold_table.css';

interface LegalHoldTableProps {
    legalHolds: LegalHold[];
    releaseLegalHold: (legalHold: LegalHold) => void;
    showUpdateModal: (legalHold: LegalHold) => void;
    showSecretModal: (legalHold: LegalHold) => void;
    runLegalHold: (id: string) => Promise<unknown>;
    refresh: () => void;
}

const HEADERS = ['Name', 'Start date', 'End date', 'Targeted users', 'Targeted groups', 'Last run', 'Actions'];

const uniqueIds = (lists: Array<string[] | null>) => {
    return Array.from(new Set(lists.flatMap((ids) => ids ?? []).filter(Boolean)));
};

const LegalHoldTable = (props: LegalHoldTableProps) => {
    const dispatch = useDispatch();
    const {legalHolds} = props;

    const userIds = useMemo(() => uniqueIds(legalHolds.map((lh) => lh.user_ids)), [legalHolds]);
    const groupIds = useMemo(() => uniqueIds(legalHolds.map((lh) => lh.group_ids)), [legalHolds]);

    useEffect(() => {
        dispatch(getMissingProfilesByIds(userIds));
        dispatch(getMissingGroupsByIds(groupIds));
    }, [dispatch, userIds, groupIds]);

    return (
        <div
            role='table'
            aria-label='Legal holds table'
            className='legal-hold-table'
        >
            <div
                role='row'
                className='legal-hold-table__row'
            >
                {HEADERS.map((header) => (
                    <div
                        key={header}
                        role='columnheader'
                        className='legal-hold-table__header'
                    >
                        {header}
                    </div>
                ))}
            </div>
            {legalHolds.map((legalHold) => (
                <LegalHoldRow
                    legalHold={legalHold}
                    key={'legalhold_' + legalHold.id}
                    releaseLegalHold={props.releaseLegalHold}
                    showUpdateModal={props.showUpdateModal}
                    showSecretModal={props.showSecretModal}
                    runLegalHold={props.runLegalHold}
                    refresh={props.refresh}
                />
            ))}
        </div>
    );
};

export default LegalHoldTable;
