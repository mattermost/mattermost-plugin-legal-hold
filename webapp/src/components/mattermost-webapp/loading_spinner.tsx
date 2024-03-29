// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import LocalizedIcon from '@/components/mattermost-webapp/localized_icon';

type Props = {
    text: React.ReactNode;
    style?: React.CSSProperties;
}

// eslint-disable-next-line react/prefer-stateless-function
export default class LoadingSpinner extends React.PureComponent<Props> {
    public static defaultProps: Props = {
        text: null,
    };

    public render() {
        return (
            <span
                id='loadingSpinner'
                className={'LoadingSpinner' + (this.props.text ? ' with-text' : '')}
                style={this.props.style}
                data-testid='loadingSpinner'
            >
                <LocalizedIcon
                    className='fa fa-spinner fa-fw fa-pulse spinner'
                    component='span'
                    title='Loading Icon'
                />
                {this.props.text}
            </span>
        );
    }
}
