import React, {useEffect} from 'react';

import LegalHoldRow from '@/components/legal_hold_table/legal_hold_row';
import {LegalHold} from '@/types';

interface LegalHoldTableProps {
    legalHolds: LegalHold[];
    actions: {
        getMissingProfilesByIds: Function,
    },
    releaseLegalHold: Function,
    showUpdateModal: Function,
    runLegalHold: Function,
    showSecretModal: Function,
}

const LegalHoldTable = (props: LegalHoldTableProps) => {
    const legalHolds = props.legalHolds;

    const user_ids = Array.from(
        new Set(
            legalHolds.map((lh) => lh.user_ids).filter((i) => i !== null).reduce((prev, cur) => prev.concat(cur), []).filter((i) => i !== null),
        ),
    );

    useEffect(() => {
        props.actions.getMissingProfilesByIds(
            user_ids,
        );
    }, [props.actions, user_ids]);

    return (
        <div>
            <div
                aria-label='Legal Holds Table'
                style={{
                    display: 'grid',
                    gridTemplateColumns: 'auto auto auto auto auto',
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
                >{'Users'}</div>
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
                            runLegalHold={props.runLegalHold}
                            showSecretModal={props.showSecretModal}
                        />
                    );
                })}
            </div>
        </div>
    );
};

export default LegalHoldTable;
