// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {FormattedMessage} from 'react-intl';

import LoadingWrapper from '@/components/mattermost-webapp/loading_wrapper';

type Props = {
    saving: boolean;
    type: 'submit' | 'button';
    disabled?: boolean;
    id?: string;
    onClick?: (e: React.MouseEvent<HTMLButtonElement>) => void;
    savingMessage?: React.ReactNode;
    defaultMessage?: React.ReactNode;
    btnClass?: string;
    extraClasses?: string;
}

// eslint-disable-next-line react/prefer-stateless-function
export default class SaveButton extends React.PureComponent<Props> {
    public static defaultProps: Partial<Props> = {
        type: 'submit',
        btnClass: '',
        defaultMessage: (
            <FormattedMessage
                id='save_button.save'
                defaultMessage='Save'
            />
        ),
        disabled: false,
        extraClasses: '',
        savingMessage: (
            <FormattedMessage
                id='save_button.saving'
                defaultMessage='Saving'
            />
        ),
    };

    public render() {
        const {
            saving,
            disabled,
            savingMessage,
            defaultMessage,
            btnClass,
            extraClasses,
            type,
            ...props
        } = this.props;

        let className = 'btn';
        if (!btnClass) {
            className += ' btn-primary';
        }

        if (!disabled || saving) {
            className += ' ' + btnClass;
        }

        if (extraClasses) {
            className += ' ' + extraClasses;
        }

        return (
            <button
                type={type}
                className={className}
                disabled={disabled}
                {...props}
            >
                <LoadingWrapper
                    loading={saving}
                    text={savingMessage}
                >
                    <span>{defaultMessage}</span>
                </LoadingWrapper>
            </button>
        );
    }
}
