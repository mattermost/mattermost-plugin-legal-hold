import React, {useEffect} from 'react';

import LegalHoldRow from '@/components/legal_hold_table/legal_hold_row';
import {LegalHold} from '@/types';

interface LegalHoldTableProps {
    legalHolds: LegalHold[];
    actions: {
        getMissingProfilesByIds: Function,
        getMissingGroupsByIds: Function,
    },
    releaseLegalHold: Function,
    showUpdateModal: Function,
    showSecretModal: Function,
    runLegalHold: (id: string) => Promise<void>;
    refresh: () => void;
}

const LegalHoldTable = (props: LegalHoldTableProps) => {
    const legalHolds = props.legalHolds;

    const user_ids = Array.from(
        new Set(
            legalHolds.
                map((lh) => lh.user_ids). // Put each LH's array of user IDs into an array
                filter((i) => i !== null). // Drop any arrays that are null
                reduce((prev, cur) => prev.concat(cur), []). // Flatten the list into a single array
                filter((i) => i !== null), // Drop any IDs that are null
        ),
    );

    const group_ids = Array.from(
        new Set(
            legalHolds.
                map((lh) => lh.group_ids). // Put each LH's array of group IDs into an array
                filter((i) => i !== null). // Drop any arrays that are null
                reduce((prev, cur) => prev.concat(cur), []). // Flatten the list into a single array
                filter((i) => i !== null), // Drop any IDs that are null
        ),
    );

    useEffect(() => {
        props.actions.getMissingProfilesByIds(
            user_ids,
        );
        props.actions.getMissingGroupsByIds(
            group_ids,
        );
    }, [props.actions, user_ids, group_ids]);

    return (
        <div>
            <div
                aria-label='Legal Holds Table'
                style={{
                    display: 'grid',
                    gridTemplateColumns: 'auto auto auto auto auto auto auto',
                    columnGap: '10px',
                    rowGap: '10px',
                    alignItems: 'center',
                }}
            >
                <div
                    aria-label='name header'
                    style={{fontWeight: 'bold'}}
                >{'Name'}</div>
                <div
                    aria-label='start date header'
                    style={{fontWeight: 'bold'}}
                >{'Start Date'}</div>
                <div
                    aria-label='end date header'
                    style={{fontWeight: 'bold'}}
                >{'End Date'}</div>
                <div
                    aria-label='users header'
                    style={{fontWeight: 'bold'}}
                >{'Targeted Users'}</div>
                <div
                    aria-label='groups header'
                    style={{fontWeight: 'bold'}}
                >{'Targeted Groups'}</div>
                <div
                    aria-label='actions header'
                    style={{fontWeight: 'bold'}}
                >{'Last Run'}</div>
                <div
                    aria-label='actions header'
                    style={{fontWeight: 'bold'}}
                >{'Actions'}</div>
                {legalHolds.map((legalHold) => {
                    return (
                        <LegalHoldRow
                            legalHold={legalHold}
                            key={'legalhold_' + legalHold.id}
                            releaseLegalHold={props.releaseLegalHold}
                            showUpdateModal={props.showUpdateModal}
                            showSecretModal={props.showSecretModal}
                            runLegalHold={(id: string) => props.runLegalHold(id)}
                            refresh={props.refresh}
                        />
                    );
                })}
            </div>
        </div>
    );
};

export default LegalHoldTable;
