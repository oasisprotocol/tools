#!/usr/bin/env python3
"""Incentives."""
import math
import pandas as pd
import dash_loading_spinners as dls
import dash_bootstrap_components as dbc
import plotly.express as px

from dash.dependencies import Input, Output
from dash import Dash, html, dcc, callback

app = Dash(
    __name__,
    external_stylesheets=[dbc.themes.LUX, dbc.icons.BOOTSTRAP],
)
server = app.server

def estimate(df, percent, common_pool, staked_tokens):
    """estimate
    
    Based on given parameters estimate how long it will take for the common 
    pool to be depleated.
    
    :param df: DataFrame with current staking rewards schedule
    :param percent: Number between 0-20 that represents staking rewards 
    :param common_pool: Number of tokens that are left in the common pool 
    :param staked_tokens: Number of tokens that are now delegated 

    :Returns:
        DataFrame: with new rewards schedule
        Int: of epochs until common pool depleation
    """

    today = pd.to_datetime("today").date()

    # Estimate before locking rewards
    cp_df = df[(df["Estimated date"] > today) & (
        df["Estimated annualized rewards %"] > percent)].copy()
    # create daily rewards and store them in factor attribute
    cp_df.loc[:, "factor"] = (cp_df["Reward per epoch %"]/100 + 1)**24
    # multiply daily rewards between each other
    val = cp_df["factor"].prod()
    # substract used rewards from common pool
    common_pool = common_pool - common_pool * (val - 1)
    # number of used epochs before locking rewards
    epochs_before = len(cp_df)*24

    # Estimate after locking rewards
    ap_df = df.copy()
    # lock Reward per epoch %
    ap_df.loc[(ap_df["Estimated annualized rewards %"] <= percent) & (ap_df["Estimated date"] > today),
           "Reward per epoch %"] = ap_df[ap_df["Estimated annualized rewards %"] <= percent]["Reward per epoch %"].max()
    # lock lock Estimated annualized rewards %
    ap_df.loc[(ap_df["Estimated annualized rewards %"] <= percent) & (ap_df["Estimated date"] > today),
           "Estimated annualized rewards %"] = ap_df[ap_df["Estimated annualized rewards %"] <= percent]["Estimated annualized rewards %"].max()
    # calculate base for the logarithm
    base = 1 + (ap_df[ap_df["Estimated annualized rewards %"] <=
                percent]["Reward per epoch %"].max() / 100)
    # calculate number of epochs until common pool depletion
    epochs_after = math.log(
        (staked_tokens + common_pool) / staked_tokens, base)

    # Add number of epochs
    epochs = epochs_before + epochs_after

    return ap_df, epochs

def layout():
    """layout"""

    return dbc.Container([
        html.Div([
            html.Br(),
            html.H1(children='Estimation of Common Pool Token Depletion'),
            html.Br(),
            html.H5(id="", children='Set parameters'),
            html.Div([
                html.P(
                    'Final Fixed Annual Rewards (in %):',
                    style={'display': 'inline-block',
                           'margin-right': 20, 'width': 320}
                ),
                dcc.Input(
                    id='input_percent',
                    type='number',
                    min=0, max=20, step=0.01, value=2),
            ]),
            html.Div([
                html.P(
                    'Common Pool (in ROSE):',
                    style={'display': 'inline-block',
                           'margin-right': 20, 'width': 320}
                ),
                dcc.Input(
                    id='input_common_pool',
                    type='number',
                    min=0, step=1, value=741546689),
            ]),
            html.Div([
                html.P(
                    'Total tokens staked (in ROSE):',
                    style={'display': 'inline-block',
                           'margin-right': 20, 'width': 320}
                ),
                dcc.Input(
                    id='input_staked_tokens',
                    type='number',
                    min=0, step=1, value=4653472533),
            ]),
            html.Br(),
            html.H5(id="", children='Estimate a Validator\'s Earnings'),
            html.Div([
                html.P(
                    'ROSE price (in $):',
                    style={'display': 'inline-block',
                           'margin-right': 20, 'width': 320}
                ),
                dcc.Input(
                    id='input_rose_price',
                    type='number',
                    min=0, step=0.0000001, value=0.04),
            ]),
            html.Div([
                html.P(
                    'Delegations to Validator (in ROSE):',
                    style={'display': 'inline-block',
                           'margin-right': 20, 'width': 320}
                ),
                dcc.Input(
                    id='input_validator_stake',
                    type='number',
                    min=0, step=1, value=20000000),
            ]),
            html.Div([
                html.P(
                    'Validator commission (in %):',
                    style={'display': 'inline-block',
                           'margin-right': 20, 'width': 320}
                ),
                dcc.Input(
                    id='input_commission',
                    type='number',
                    min=0, max=100, step=1, value=20),
            ]),
            html.Br(),
            html.Br(),
            html.H5(id="labelinfo", children=''),
            html.Br(),
            html.H5(id="validator_earnings", children=''),
            dls.Hash(
                dcc.Graph(id="linegraph"),
                color="#3292a8",
                speed_multiplier=2,
                size=100,
            ),
            dbc.Alert(
                html.Center(
                    html.B("We are assuming 1 epoch per hour and that a day has 24 epochs.")),
                id="alert-fade",
                dismissable=True,
                is_open=True,
            ),
        ])
    ])

@callback(
    [
        Output(component_id='labelinfo', component_property='children'),
        Output(component_id='validator_earnings',
               component_property='children'),
        Output(component_id='linegraph', component_property='figure')
    ],
    [
        Input(component_id='input_common_pool', component_property='value'),
        Input(component_id='input_percent', component_property='value'),
        Input(component_id='input_staked_tokens', component_property='value'),
        Input(component_id='input_rose_price', component_property='value'),
        Input(component_id='input_validator_stake', component_property='value'),
        Input(component_id='input_commission', component_property='value')
    ]
)
def update_estimation(input_common_pool, input_percent, input_staked_tokens,
    input_rose_price, input_validator_stake, input_commission):
    """calback function update_estimation"""

    # Get and prepare dataframe
    df = pd.read_csv("staking_rewards.csv", parse_dates=True)
    df["Estimated date"] = pd.to_datetime(df["Estimated date"])
    df = df.set_index('Estimated date').asfreq(
        'D', method='ffill').reset_index()
    df["Estimated date"] = df["Estimated date"].dt.date

    if input_common_pool is None or input_percent is None or \
    input_staked_tokens is None or input_rose_price is None or \
    input_validator_stake is None or input_commission is None or \
    input_staked_tokens < input_validator_stake:
        return("Some numbers are out of bounds.", "", {})

    input_commission *= 0.01
    updated_df, epochs = estimate(
        df, input_percent, input_common_pool, input_staked_tokens)
    fig = px.line(
        updated_df,
        x="Estimated date",
        y="Estimated annualized rewards %",
        title="Staking Rewards Schedule",
    )

    estimated_rewards = updated_df["Estimated annualized rewards %"].iloc[-1]
    rewards = (0.01*estimated_rewards/12) * input_validator_stake * \
        input_rose_price * input_commission

    output_labelinfo = [
        html.Span("{:,.2f} YEARS ".format(
            epochs/24/365), style={"color": "red"}),
        html.Span("Until Common Pool depletion"),
    ]

    output_validator_earnings = [
        html.Span("${:,.2f} ".format(rewards), style={"color": "red"}),
        html.Span("is estimated Validator\'s monthly earnings when rewards drop to {:,.2f}%: ".format(
            estimated_rewards)),
    ]

    return(output_labelinfo, output_validator_earnings, fig)

app.layout = layout()

if __name__ == '__main__':
    app.run(debug=False)
