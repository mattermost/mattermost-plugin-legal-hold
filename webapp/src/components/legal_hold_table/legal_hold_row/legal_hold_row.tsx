import React from 'react';
import {UserProfile} from 'mattermost-redux/types/users';

import {LegalHold} from '@/types';
import Client from '@/client';

interface LegalHoldRowProps {
    legalHold: LegalHold;
    users: UserProfile[];
    releaseLegalHold: Function;
    showUpdateModal: Function;
}

const LegalHoldRow = (props: LegalHoldRowProps) => {
    const lh = props.legalHold;
    const startsAt = (new Date(lh.starts_at)).toLocaleDateString();
    const endsAt = lh.ends_at === 0 ? 'Never' : (new Date(lh.ends_at)).toLocaleDateString();

    const release = () => {
        props.releaseLegalHold(lh);
    };

    const usernames = props.users.map((user) => {
        if (user) {
            return `@${user.username} `;
        }
        return 'loading...';
    });

    const downloadUrl = Client.downloadUrl(lh.id);

    return (
        <React.Fragment>
            <div>{lh.display_name}</div>
            <div>{startsAt}</div>
            <div>{endsAt}</div>
            <div>{props.users.length} {'users'}</div>
            <div>
                <a
                    href='#'
                    onClick={() => props.showUpdateModal(lh)}
                >
                    {'Edit'}
                </a>
                {' '}
                <a href={downloadUrl}>{'Download'}</a>
                {' '}
                <a
                    href='#'
                    onClick={release}
                >{'Release'}</a>
            </div>
        </React.Fragment>
    );
};

export default LegalHoldRow;
